const state = {
  loading: false,
  scanning: false,
  savingConfig: false,
  lastReport: null,
  config: null,
};

const els = {
  updatedAt: document.querySelector("#updatedAt"),
  refreshBtn: document.querySelector("#refreshBtn"),
  scanBtn: document.querySelector("#scanBtn"),
  configBtn: document.querySelector("#configBtn"),
  configDialog: document.querySelector("#configDialog"),
  closeConfigBtn: document.querySelector("#closeConfigBtn"),
  configForm: document.querySelector("#configForm"),
  configPath: document.querySelector("#configPath"),
  addressesInput: document.querySelector("#addressesInput"),
  usernameInput: document.querySelector("#usernameInput"),
  passwordInput: document.querySelector("#passwordInput"),
  containersInput: document.querySelector("#containersInput"),
  patternsInput: document.querySelector("#patternsInput"),
  runtimeDocker: document.querySelector("#runtimeDocker"),
  runtimePodman: document.querySelector("#runtimePodman"),
  configMessage: document.querySelector("#configMessage"),
  configuredPCs: document.querySelector("#configuredPCs"),
  discoveryState: document.querySelector("#discoveryState"),
  scanState: document.querySelector("#scanState"),
  targets: document.querySelector("#targets"),
  scanResults: document.querySelector("#scanResults"),
  nosanaMatches: document.querySelector("#nosanaMatches"),
  containersSeen: document.querySelector("#containersSeen"),
  runtimesAvailable: document.querySelector("#runtimesAvailable"),
  targetsScanned: document.querySelector("#targetsScanned"),
};

els.refreshBtn.addEventListener("click", () => refresh());
els.scanBtn.addEventListener("click", () => scanLAN());
els.configBtn.addEventListener("click", () => openConfig());
els.closeConfigBtn.addEventListener("click", () => els.configDialog.close());
els.configForm.addEventListener("submit", (event) => {
  event.preventDefault();
  saveBulkConfig();
});

refresh();
setInterval(refresh, 10000);

async function refresh() {
  if (state.loading) return;
  state.loading = true;
  els.refreshBtn.disabled = true;
  setPill(els.discoveryState, "Loading", "");

  try {
    const report = await getJSON("/api/nosana");
    state.lastReport = report;
    renderReport(report);
    setPill(els.discoveryState, report.summary.nosanaMatches > 0 ? "Live" : "No hosts", report.summary.nosanaMatches > 0 ? "ok" : "warn");
  } catch (error) {
    els.targets.innerHTML = `<div class="error-text">${escapeHTML(error.message)}</div>`;
    setPill(els.discoveryState, "Error", "error");
  } finally {
    state.loading = false;
    els.refreshBtn.disabled = false;
  }
}

async function scanLAN() {
  if (state.scanning) return;
  state.scanning = true;
  els.scanBtn.disabled = true;
  setPill(els.scanState, "Scanning", "");

  try {
    const report = await getJSON("/api/pc/scan");
    renderScan(report);
    setPill(els.scanState, `${report.results?.length || 0} found`, report.results?.length ? "ok" : "muted");
  } catch (error) {
    els.scanResults.innerHTML = `<div class="error-text">${escapeHTML(error.message)}</div>`;
    setPill(els.scanState, "Error", "error");
  } finally {
    state.scanning = false;
    els.scanBtn.disabled = false;
  }
}

async function getJSON(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: { Accept: "application/json", ...(options.headers || {}) },
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(body.error || `${response.status} ${response.statusText}`);
  }
  return body;
}

async function openConfig() {
  els.configDialog.showModal();
  els.configMessage.textContent = "Loading current configuration.";
  await loadConfig();
}

async function loadConfig() {
  try {
    const response = await getJSON("/api/config");
    state.config = response;
    renderConfig(response);
    els.configMessage.textContent = "Ready.";
  } catch (error) {
    els.configMessage.innerHTML = `<span class="error-text">${escapeHTML(error.message)}</span>`;
  }
}

async function saveBulkConfig() {
  if (state.savingConfig) return;
  state.savingConfig = true;
  const runtimes = [];
  if (els.runtimeDocker.checked) runtimes.push("docker");
  if (els.runtimePodman.checked) runtimes.push("podman");

  const payload = {
    addresses: els.addressesInput.value,
    username: els.usernameInput.value,
    password: els.passwordInput.value,
    runtimes,
    containerNames: splitCSV(els.containersInput.value),
    containerPatterns: splitCSV(els.patternsInput.value),
    maxHosts: 1024,
  };

  try {
    const response = await getJSON("/api/config/pcs/bulk", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    renderConfig({ configPath: response.configPath, config: response.config });
    const warning = (response.warnings || []).join(" ");
    els.configMessage.textContent = `Saved ${response.added} new PCs and updated ${response.updated}. ${warning}`.trim();
    els.passwordInput.value = "";
    await refresh();
  } catch (error) {
    els.configMessage.innerHTML = `<span class="error-text">${escapeHTML(error.message)}</span>`;
  } finally {
    state.savingConfig = false;
  }
}

function renderConfig(response) {
  const cfg = response.config || {};
  const pcs = cfg.pcs || [];
  els.configPath.textContent = response.configPath || "Config path unknown";

  if (!pcs.length) {
    els.configuredPCs.innerHTML = `<div class="empty">No PCs configured yet.</div>`;
    return;
  }

  els.configuredPCs.innerHTML = pcs.map((pc) => `
    <article class="candidate">
      <strong>${escapeHTML(pc.name)}</strong>
      <div class="meta">${escapeHTML([pc.address, pc.sshTarget].filter(Boolean).join(" | "))}</div>
      <div class="meta">containers: ${escapeHTML((pc.containerNames || []).join(", ") || "default patterns")}</div>
      <div class="meta">patterns: ${escapeHTML((pc.containerPatterns || []).join(", ") || (cfg.defaultContainerPatterns || []).join(", "))}</div>
    </article>
  `).join("");
}

function renderReport(report) {
  const summary = report.summary || {};
  els.nosanaMatches.textContent = summary.nosanaMatches ?? 0;
  els.containersSeen.textContent = summary.containersSeen ?? 0;
  els.runtimesAvailable.textContent = summary.runtimesAvailable ?? 0;
  els.targetsScanned.textContent = summary.targetsScanned ?? 0;
  els.updatedAt.textContent = `Updated ${formatTime(report.generatedAt)}`;

  const targets = report.targets || [];
  if (!targets.length) {
    els.targets.innerHTML = `<div class="empty">No targets reported.</div>`;
    return;
  }
  els.targets.innerHTML = targets.map(renderTarget).join("");
}

function renderTarget(target) {
  const label = target.skipped ? "Skipped" : target.scope || "target";
  const pillClass = target.skipped ? "warn" : "muted";
  const meta = [target.address, target.sshTarget].filter(Boolean).join(" | ") || target.scope || "";
  const runtimes = target.skipped
    ? `<div class="meta">${escapeHTML(target.skipReason || "")}</div>`
    : (target.runtimes || []).map(renderRuntime).join("");

  return `
    <article class="target">
      <div class="target-head">
        <div class="target-title">
          <strong>${escapeHTML(target.name)}</strong>
          <div class="meta">${escapeHTML(meta)}</div>
        </div>
        <span class="pill ${pillClass}">${escapeHTML(label)}</span>
      </div>
      ${runtimes || `<div class="empty">No runtime data.</div>`}
    </article>
  `;
}

function renderRuntime(runtime) {
  if (!runtime.available) {
    return `
      <div class="runtime">
        <div class="runtime-name">${escapeHTML(runtime.type)} <span class="pill warn">Unavailable</span></div>
        <div class="meta">${escapeHTML(runtime.error || "")}</div>
      </div>
    `;
  }

  const containers = runtime.containers || [];
  return `
    <div class="runtime">
      <div class="runtime-name">${escapeHTML(runtime.type)} <span class="pill ok">Available</span></div>
      <div class="containers">
        ${containers.length ? containers.map(renderContainer).join("") : `<div class="empty">No containers running.</div>`}
      </div>
    </div>
  `;
}

function renderContainer(container) {
  const matched = container.matched ? `<span class="pill ok">Nosana</span>` : `<span class="pill muted">Container</span>`;
  const nested = (container.nested || []).length
    ? `<div class="nested">${container.nested.map(renderContainer).join("")}</div>`
    : "";

  return `
    <div>
      <div class="container-row">
        <span><strong>${escapeHTML(container.name || container.id || "unnamed")}</strong></span>
        <span>${escapeHTML(container.image || "")}</span>
        <span>${matched}</span>
      </div>
      ${container.status ? `<div class="meta">${escapeHTML(container.status)}</div>` : ""}
      ${nested}
    </div>
  `;
}

function renderScan(report) {
  const results = report.results || [];
  if (!results.length) {
    els.scanResults.innerHTML = `<div class="empty">No candidates found.</div>`;
    return;
  }
  els.scanResults.innerHTML = results.map((candidate) => `
    <article class="candidate">
      <div class="row">
        <strong>${escapeHTML(candidate.ip)}</strong>
        <span class="pill">${(candidate.openPorts || []).map((port) => escapeHTML(String(port))).join(", ")}</span>
      </div>
    </article>
  `).join("");
}

function setPill(element, text, className) {
  element.className = `pill ${className || ""}`.trim();
  element.textContent = text;
}

function splitCSV(value) {
  return String(value || "")
    .split(",")
    .map((part) => part.trim())
    .filter(Boolean);
}

function formatTime(value) {
  if (!value) return "unknown";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
