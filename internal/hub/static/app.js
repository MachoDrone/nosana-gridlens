const state = {
  loading: false,
  scanning: false,
  savingConfig: false,
  lastReport: null,
  config: null,
  targetSort: storageGet("gridlens.targetSort") || "ip",
  theme: storageGet("gridlens.theme") || "dark",
};

const els = {
  updatedAt: document.querySelector("#updatedAt"),
  hubIdentity: document.querySelector("#hubIdentity"),
  themeBtn: document.querySelector("#themeBtn"),
  refreshBtn: document.querySelector("#refreshBtn"),
  scanBtn: document.querySelector("#scanBtn"),
  configBtn: document.querySelector("#configBtn"),
  sortNameBtn: document.querySelector("#sortNameBtn"),
  sortIPBtn: document.querySelector("#sortIPBtn"),
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
  dimmedScanDetails: document.querySelector("#dimmedScanDetails"),
  dimmedScanSummary: document.querySelector("#dimmedScanSummary"),
  dimmedScanResults: document.querySelector("#dimmedScanResults"),
  nosanaMatches: document.querySelector("#nosanaMatches"),
  containersSeen: document.querySelector("#containersSeen"),
  pcCount: document.querySelector("#pcCount"),
  gridLensTraffic: document.querySelector("#gridLensTraffic"),
  runtimesAvailable: document.querySelector("#runtimesAvailable"),
};

els.themeBtn.addEventListener("click", () => toggleTheme());
els.refreshBtn.addEventListener("click", () => refresh());
els.scanBtn.addEventListener("click", () => scanLAN());
els.configBtn.addEventListener("click", () => openConfig());
els.sortNameBtn.addEventListener("click", () => setTargetSort("name"));
els.sortIPBtn.addEventListener("click", () => setTargetSort("ip"));
els.closeConfigBtn.addEventListener("click", () => els.configDialog.close());
els.configForm.addEventListener("submit", (event) => {
  event.preventDefault();
  saveBulkConfig();
});

applyTheme(state.theme);
refresh();
updateSortButtons();
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
    const hostCount = report.summary.nosanaHosts ?? report.summary.nosanaMatches ?? 0;
    setPill(els.discoveryState, hostCount > 0 ? "Live" : "No hosts", hostCount > 0 ? "ok" : "warn");
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
      <div class="meta">${escapeHTML(pcConfigMeta(pc))}</div>
      <div class="meta">containers: ${escapeHTML((pc.containerNames || []).join(", ") || "default patterns")}</div>
      <div class="meta">patterns: ${escapeHTML((pc.containerPatterns || []).join(", ") || (cfg.defaultContainerPatterns || []).join(", "))}</div>
    </article>
  `).join("");
}

function renderReport(report) {
  const summary = report.summary || {};
  els.nosanaMatches.textContent = summary.nosanaHosts ?? summary.nosanaMatches ?? 0;
  els.containersSeen.textContent = summary.containersSeen ?? 0;
  els.pcCount.textContent = summary.pcCount ?? configuredTargets(report).length;
  els.gridLensTraffic.textContent = formatTrafficRate(summary.gridLensTraffic);
  els.gridLensTraffic.title = trafficTitle(summary.gridLensTraffic);
  els.runtimesAvailable.textContent = summary.runtimesAvailable ?? 0;
  els.updatedAt.textContent = `Updated ${formatTime(report.generatedAt)}`;
  els.hubIdentity.textContent = hubIdentityText(report.hub);

  const targets = sortedTargets(report.targets || []);
  if (!targets.length) {
    els.targets.innerHTML = `<div class="empty">No targets reported.</div>`;
    return;
  }
  els.targets.innerHTML = targets.map(renderTarget).join("");
}

function renderTarget(target) {
  const hostCount = countHostContainers(target);
  if (target.scope === "local" && hostCount === 0) {
    return renderCollapsedLocal(target);
  }

  const label = target.skipped ? "Credentials" : hostCount > 0 ? `${hostCount} host${hostCount === 1 ? "" : "s"}` : "No host";
  const pillClass = target.skipped ? "warn" : hostCount > 0 ? "ok" : "muted";
  const meta = pcMeta(target);
  const displayName = targetDisplayName(target);
  const visibleRuntimes = visibleRuntimeReports(target);
  const runtimes = target.skipped
    ? `<div class="meta">${escapeHTML(target.skipReason || "")}</div>`
    : visibleRuntimes.map(renderRuntime).join("");

  return `
    <article class="target">
      <div class="target-head">
        <div class="target-title">
          <strong>${escapeHTML(displayName)}</strong>
          <div class="meta">${escapeHTML(meta)}</div>
        </div>
        <span class="pill ${pillClass}">${escapeHTML(label)}</span>
      </div>
      ${runtimes || `<div class="empty">No runtime data.</div>`}
    </article>
  `;
}

function renderCollapsedLocal(target) {
  const displayName = targetDisplayName(target);
  return `
    <details class="target target-collapsed">
      <summary>
        <span>
          <strong>${escapeHTML(displayName)}</strong>
          <span class="meta">No Nosana host detected on this Hub PC</span>
        </span>
        <span class="pill muted">Expand</span>
      </summary>
      ${visibleRuntimeReports(target).map(renderRuntime).join("") || `<div class="empty">No local runtime data.</div>`}
    </details>
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
    els.dimmedScanDetails.hidden = true;
    return;
  }
  const enriched = results.map((candidate) => ({ candidate, status: candidateStatus(candidate) }));
  const active = enriched.filter((item) => !item.status.dim);
  const dimmed = enriched.filter((item) => item.status.dim);

  els.scanResults.innerHTML = active.length
    ? active.map((item) => renderCandidate(item.candidate, item.status)).join("")
    : `<div class="empty">No scanned candidates currently have discoverable Nosana hosts.</div>`;

  els.dimmedScanDetails.hidden = dimmed.length === 0;
  els.dimmedScanSummary.textContent = `${dimmed.length} candidate${dimmed.length === 1 ? "" : "s"} without Nosana hosts or credentials`;
  els.dimmedScanResults.innerHTML = dimmed.map((item) => renderCandidate(item.candidate, item.status)).join("");
}

function setTargetSort(sort) {
  state.targetSort = sort;
  storageSet("gridlens.targetSort", sort);
  updateSortButtons();
  if (state.lastReport) renderReport(state.lastReport);
}

function toggleTheme() {
  applyTheme(state.theme === "dark" ? "light" : "dark");
}

function applyTheme(theme) {
  state.theme = theme === "light" ? "light" : "dark";
  document.documentElement.dataset.theme = state.theme;
  document.documentElement.style.colorScheme = state.theme;
  storageSet("gridlens.theme", state.theme);
  els.themeBtn.textContent = state.theme === "dark" ? "Light" : "Dark";
  els.themeBtn.title = state.theme === "dark" ? "Switch to light mode" : "Switch to dark mode";
}

function updateSortButtons() {
  els.sortNameBtn.classList.toggle("active", state.targetSort === "name");
  els.sortIPBtn.classList.toggle("active", state.targetSort === "ip");
}

function sortedTargets(targets) {
  const configured = targets.filter((target) => target.scope === "configured");
  const local = targets.filter((target) => target.scope !== "configured");
  configured.sort((a, b) => {
    if (state.targetSort === "name") {
      return targetDisplayName(a).localeCompare(targetDisplayName(b), undefined, { numeric: true, sensitivity: "base" }) || compareIP(a.address, b.address);
    }
    return compareIP(a.address, b.address) || targetDisplayName(a).localeCompare(targetDisplayName(b), undefined, { numeric: true, sensitivity: "base" });
  });
  const localWithHosts = local.filter((target) => countHostContainers(target) > 0);
  const localWithoutHosts = local.filter((target) => countHostContainers(target) === 0);
  if (!localWithHosts.length) {
    return [...configured, ...localWithoutHosts];
  }
  return [...configured, ...localWithHosts].sort((a, b) => {
    if (state.targetSort === "name") {
      return targetDisplayName(a).localeCompare(targetDisplayName(b), undefined, { numeric: true, sensitivity: "base" }) || compareIP(a.address, b.address);
    }
    return compareIP(a.address, b.address) || targetDisplayName(a).localeCompare(targetDisplayName(b), undefined, { numeric: true, sensitivity: "base" });
  });
}

function configuredTargets(report) {
  return (report.targets || []).filter((target) => target.scope === "configured");
}

function visibleRuntimeReports(target) {
  const runtimes = target.runtimes || [];
  const hasNestedPodman = runtimes.some((runtime) =>
    runtime.type === "docker" && (runtime.containers || []).some((container) => (container.nested || []).length > 0)
  );
  return runtimes.filter((runtime) => !(runtime.type === "podman" && !runtime.available && hasNestedPodman));
}

function pcMeta(target) {
  const parts = [];
  if (target.address) parts.push(target.address);
  if (target.sshTarget) parts.push(sshLoginSummary(target.sshTarget));
  if (!target.sshTarget && target.scope === "configured") parts.push("no SSH login");
  if (!parts.length) parts.push(target.scope || "target");
  return parts.join(" | ");
}

function pcConfigMeta(pc) {
  const parts = [];
  if (pc.address) parts.push(pc.address);
  if (pc.sshTarget) parts.push(sshLoginSummary(pc.sshTarget));
  if (!pc.sshTarget) parts.push("no SSH login");
  return parts.join(" | ");
}

function sshLoginSummary(sshTarget) {
  const login = String(sshTarget || "").split("@")[0].trim();
  return `SSH ${login || "login"}/key`;
}

function targetDisplayName(target) {
  return target.hostName || target.name || target.address || "unknown PC";
}

function hubIdentityText(hub) {
  if (!hub) return "Hub PC unknown";
  const name = hub.name || "unknown";
  const ips = (hub.ipAddresses || []).join(", ") || "no private LAN IP detected";
  return `Hub PC ${name} | IP ${ips}`;
}

function countHostContainers(target) {
  let count = 0;
  for (const runtime of target.runtimes || []) {
    for (const container of runtime.containers || []) {
      count += countHostContainersInContainer(container);
    }
  }
  return count;
}

function countHostContainersInContainer(container) {
  let count = container.matched ? 1 : 0;
  for (const nested of container.nested || []) {
    count += countHostContainersInContainer(nested);
  }
  return count;
}

function candidateStatus(candidate) {
  const report = state.lastReport || {};
  const target = (report.targets || []).find((item) => item.address === candidate.ip);
  if (!target) {
    return { dim: true, note: "Not configured yet; login credentials are needed before GridLens can inspect Nosana hosts." };
  }
  if (target.skipped) {
    return { dim: true, note: target.skipReason || "Login credentials are missing." };
  }
  const hostCount = countHostContainers(target);
  if (hostCount === 0) {
    return { dim: true, note: "Configured, but no Nosana host is currently running or discoverable." };
  }
  return { dim: false, note: `${hostCount} Nosana host${hostCount === 1 ? "" : "s"} discovered.` };
}

function renderCandidate(candidate, status) {
  return `
    <article class="candidate ${status.dim ? "dimmed" : ""}">
      <div class="row">
        <strong>${escapeHTML(candidate.ip)}</strong>
        <span class="pill">${(candidate.openPorts || []).map((port) => escapeHTML(String(port))).join(", ")}</span>
      </div>
      <div class="meta">${escapeHTML(status.note)}</div>
    </article>
  `;
}

function compareIP(a, b) {
  const left = ipNumber(a);
  const right = ipNumber(b);
  if (left === right) return 0;
  return left < right ? -1 : 1;
}

function ipNumber(value) {
  const parts = String(value || "").split(".").map((part) => Number.parseInt(part, 10));
  if (parts.length !== 4 || parts.some((part) => Number.isNaN(part))) return Number.MAX_SAFE_INTEGER;
  return (((parts[0] * 256) + parts[1]) * 256 + parts[2]) * 256 + parts[3];
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

function formatTrafficRate(traffic) {
  const bytesPerSecond = Number(traffic?.bytesPerSecond || 0);
  return `${formatBytes(bytesPerSecond)}/s`;
}

function trafficTitle(traffic) {
  const total = Number(traffic?.totalBytes || 0);
  const windowSeconds = Number(traffic?.windowSeconds || 60);
  const sessions = Number(traffic?.sshSessions || 0);
  const checks = Number(traffic?.portChecks || 0);
  const parts = [`GridLens estimated traffic: ${formatBytes(total)} over ${windowSeconds}s`];
  if (sessions) parts.push(`${sessions} SSH sessions`);
  if (checks) parts.push(`${checks} LAN port checks`);
  return parts.join(" | ");
}

function formatBytes(value) {
  const bytes = Math.max(0, Number(value) || 0);
  if (bytes < 1024) return `${Math.round(bytes)} B`;
  const kib = bytes / 1024;
  if (kib < 1024) return `${kib.toFixed(kib >= 10 ? 0 : 1)} KB`;
  const mib = kib / 1024;
  return `${mib.toFixed(mib >= 10 ? 0 : 1)} MB`;
}

function storageGet(key) {
  try {
    return localStorage.getItem(key);
  } catch (_) {
    return null;
  }
}

function storageSet(key, value) {
  try {
    localStorage.setItem(key, value);
  } catch (_) {}
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
