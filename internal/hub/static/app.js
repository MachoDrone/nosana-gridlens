const state = {
  loading: false,
  scanning: false,
  lastReport: null,
};

const els = {
  updatedAt: document.querySelector("#updatedAt"),
  refreshBtn: document.querySelector("#refreshBtn"),
  scanBtn: document.querySelector("#scanBtn"),
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

async function getJSON(path) {
  const response = await fetch(path, { headers: { Accept: "application/json" } });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(body.error || `${response.status} ${response.statusText}`);
  }
  return body;
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
