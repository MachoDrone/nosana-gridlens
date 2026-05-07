# GridLens for Nosana Hosts — Codex Hand-off Brief

_Last updated: 2026-05-07_  
_Primary implementation target: **Option 2 — Private Remote Mode with WireGuard**_  
_Working repo name: `MachoDrone/nosana-gridlens`_  
_Working product name: **GridLens**_

This document is written for Codex and human reviewers. It is intentionally explicit because the project has several “do not break the user’s machine” constraints that are more important than fast feature delivery.

---

## 0. Codex operating instruction

Before changing code, Codex should read:

1. `AGENTS.md` at the repository root.
2. `docs/CODEX_HANDOFF.md` — this document.
3. `docs/WIREGUARD_PRIVATE_REMOTE_MODE.md`, if present.
4. `docs/REQUIREMENTS.md`, if present.
5. Existing code and tests.

If a requested implementation conflicts with a non-negotiable requirement in this document, Codex must not silently choose a different architecture. It should either implement the safer interpretation or leave a short note in the PR summary under **Blocked / needs owner decision**.

---

## 1. Product summary

**GridLens** is a free, self-hostable monitoring console for Nosana host operators. The initial build should focus on secure private remote access from a phone or PC to a local GridLens Hub using WireGuard.

The user should eventually be able to run one command from GitHub:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/MachoDrone/nosana-gridlens/main/start.sh)
```

Then GridLens should guide the user through:

1. Checking dependencies.
2. Installing missing dependencies only after explicit permission.
3. Offering targeted updates for dependencies only when safe.
4. Starting a local Hub.
5. Creating an isolated GridLens-owned WireGuard interface.
6. Adding a phone/laptop peer.
7. Showing a WireGuard QR code in the terminal.
8. Waiting for a live handshake.
9. Showing a second QR code or URL to open the GridLens web UI through the tunnel.
10. Diagnosing failures clearly.

The MVP is not yet about full Nosana telemetry. It is about perfecting the private remote access UX and safety model first.

---

## 2. Current architectural decision

Start with:

```text
GridLens Hub on user's machine
  ├─ serves local web UI on 127.0.0.1:8787
  ├─ serves tunnel-only web UI on 10.x.y.1:8787
  ├─ owns one WireGuard interface, default glwg0
  ├─ owns one server WireGuard keypair
  ├─ owns one peer record per phone/laptop
  └─ stores state in /var/lib/gridlens and config in /etc/gridlens

Phone/laptop
  └─ official WireGuard app imports one peer config from QR code
```

The initial route is **hub-only**:

```text
phone → WireGuard tunnel → GridLens Hub UI only
```

Do not route the phone to the whole LAN by default. Do not route phone internet traffic through the Hub.

---

## 3. Why WireGuard mode first

The project explored Cloudflare Workers, GitHub Pages, relay-based monitoring, Tor, Tailscale, public HTTPS, and self-hosted hubs. The first serious implementation path is **WireGuard Private Remote Mode** because it satisfies the most important user goals without requiring a SaaS account:

- no Cloudflare account;
- no GitHub Pages runtime dependency;
- no Tailscale account;
- no Nosana API key;
- no Solana RPC key;
- no hosted database;
- no hosted dashboard;
- no cloud worker;
- remote access can be private and encrypted.

Important limitation: direct WireGuard access still needs a reachable UDP path to the Hub. At least one must be true:

1. The Hub has a public IP and the UDP port is open.
2. The router forwards the selected UDP port to the Hub.
3. IPv6 is globally reachable and firewall allows the UDP port.
4. A future relay/VPS/tunnel mode is used.

GridLens must make this limitation understandable, not hide it.

---

## 4. Non-negotiable requirements

### 4.1 Do not break existing WireGuard

GridLens must treat existing WireGuard setups as read-only unless GridLens created the resource.

GridLens must not:

- edit `/etc/wireguard/wg0.conf`;
- use the generic `wg0` interface name;
- restart `wg-quick@wg0` or any non-GridLens tunnel;
- overwrite existing WireGuard private keys;
- change existing peer records;
- run `wg-quick save` on unrelated interfaces;
- change system DNS globally;
- set `AllowedIPs = 0.0.0.0/0` by default;
- route all phone traffic through the Hub;
- enable IP forwarding by default;
- enable NAT masquerading by default;
- change firewall policy without explicit user permission;
- run broad system upgrades such as `apt upgrade`.

### 4.2 GridLens-owned resources only

GridLens may create/manage only resources it owns and can prove it owns.

Default owned resources:

```text
/etc/gridlens/
/etc/gridlens/config.toml
/etc/gridlens/wireguard/server.key
/etc/gridlens/wireguard/server.pub
/etc/gridlens/wireguard/peers/*.json
/etc/wireguard/glwg0.conf
/var/lib/gridlens/
/var/lib/gridlens/gridlens.db
/var/lib/gridlens/wireguard/state.json
/var/log/gridlens/
systemd service: gridlens-hub.service
systemd service: wg-quick@glwg0.service
optional firewall rules tagged/commented as gridlens
```

If `glwg0` exists but was not created by GridLens, GridLens must not modify it. It may choose `glwg1` if free, or stop with a clear error.

### 4.3 No silent privileged changes

All privileged operations must be visible and explicit. If the process needs `sudo`, show what is being done and why.

Acceptable:

```text
WireGuard tools are missing.
GridLens can install: sudo apt-get update && sudo apt-get install -y wireguard-tools iproute2
Install now? [Y/n/show command/skip]
```

Not acceptable:

```text
Installing and upgrading packages...
```

### 4.4 Security before convenience

- Never copy, upload, print, or persist Nosana/Solana private keys.
- WireGuard server private key must be stored with `0600` permissions.
- Client/phone private keys should be displayed in the QR during pairing and should not be stored permanently by default.
- Peer revocation must remove only the peer being revoked.
- The web UI should not be exposed on `0.0.0.0` by default.
- Bind UI to `127.0.0.1` and the GridLens WireGuard IP by default.
- Treat all unauthenticated LAN/public exposure as a bug.

---

## 5. Implementation stack recommendation

For the WireGuard MVP, prefer a **Go single-binary** backend/CLI.

Rationale:

- easy one-command install later through GitHub Releases;
- no runtime Node dependency on user hosts;
- good fit for Linux service management, command execution, file permissions, SQLite, and embedded assets;
- can embed the static web UI with Go `embed`;
- easier to run as a systemd service;
- easier to keep dependency prompts limited to actual system dependencies like WireGuard tools.

Suggested stack:

```text
Language: Go
CLI: cobra or urfave/cli, or a small standard-library CLI if preferred
HTTP server: Go net/http
Database: SQLite through a well-maintained Go driver
Config: TOML or YAML; TOML preferred for small system config
QR rendering: built-in Go QR library with terminal rendering; do not require qrencode
Frontend: initially static HTML/CSS/JS embedded in Go; later React/Vite if needed
Tests: Go unit tests with fake command runner; optional integration tests gated behind root/netns availability
```

Do not block MVP on Nosana Kit or Solana Kit. Those are future data-source/enrichment layers. WireGuard mode can launch with placeholder fleet data and a Hub health page.

---

## 6. Repository layout to create

Suggested initial tree:

```text
nosana-gridlens/
  AGENTS.md
  README.md
  LICENSE
  go.mod
  go.sum
  start.sh
  cmd/
    gridlens/
      main.go
  internal/
    app/
      version.go
    cli/
      root.go
      hub.go
      phone.go
      wireguard.go
      doctor.go
    config/
      config.go
      paths.go
    deps/
      detect.go
      package_managers.go
    execx/
      runner.go
      fake_runner.go
    hub/
      server.go
      auth.go
      routes.go
      health.go
    qrcode/
      terminal.go
    systemd/
      units.go
      install.go
    wireguard/
      config.go
      keys.go
      manager.go
      peers.go
      status.go
      doctor.go
      ipalloc.go
      firewall.go
  web/
    static/
      index.html
      app.js
      styles.css
  docs/
    CODEX_HANDOFF.md
    REQUIREMENTS.md
    WIREGUARD_PRIVATE_REMOTE_MODE.md
    SECURITY.md
    TROUBLESHOOTING.md
  testdata/
    wg-show/
    ss/
  .github/
    workflows/
      test.yml
      release.yml
```

The first PR can be smaller, but do not put everything into one untestable shell script.

---

## 7. Command surface

MVP commands:

```bash
gridlens version
gridlens setup wireguard
gridlens hub start
gridlens hub status
gridlens phone add "Alice iPhone"
gridlens phone list
gridlens phone revoke "Alice iPhone"
gridlens phone rotate "Alice iPhone"
gridlens wireguard status
gridlens doctor wireguard
gridlens wireguard disable
gridlens uninstall --dry-run
gridlens uninstall
```

Nice-to-have later:

```bash
gridlens setup hub
gridlens setup service
gridlens ui pair
gridlens ui open
gridlens deps check
gridlens deps install
gridlens firewall suggest
gridlens router suggest
gridlens agent join
gridlens fleet status
```

---

## 8. UX flows

### 8.1 First launch

Expected user flow:

```text
Welcome to GridLens.

This setup will create a private local monitor and an optional WireGuard tunnel
for phone/PC access. GridLens will not modify existing WireGuard tunnels.

Detected:
  OS: Ubuntu 24.04
  Architecture: amd64
  systemd: yes
  Existing WireGuard interfaces: wg0 active, corpvpn active

GridLens will create a separate interface:
  glwg0

Continue? [Y/n]
```

### 8.2 Dependency check

Must identify exact missing commands:

```text
Dependency check:
  wg: missing
  wg-quick: missing
  ip: found /usr/sbin/ip
  ss: found /usr/bin/ss
  systemctl: found /usr/bin/systemctl

Required package on this system:
  wireguard-tools

Install now?
  Y: install with apt-get
  n: skip
  s: show command
```

If existing WireGuard is active and package update is available, default to **not updating**:

```text
wireguard-tools is installed.
A newer version may be available from apt.

Existing active WireGuard interfaces were detected:
  wg0
  corpvpn

GridLens recommends skipping updates now to avoid disrupting existing tunnels.
Update wireguard-tools anyway? [y/N/show command]
```

### 8.3 WireGuard setup

```text
Creating GridLens WireGuard tunnel:
  Interface: glwg0
  Hub tunnel IP: 10.94.23.1
  UDP port: 51871
  Config: /etc/wireguard/glwg0.conf

No existing GridLens config was found.
Create new GridLens tunnel? [Y/n]
```

If port conflict:

```text
UDP 51871 is already in use.
GridLens selected UDP 51872 instead.
Use 51872? [Y/n/custom]
```

### 8.4 Phone add

```bash
gridlens phone add "Alice iPhone"
```

Output:

```text
Adding phone peer: Alice iPhone

Assigned tunnel IP: 10.94.23.2
Allowed access: GridLens Hub only, 10.94.23.1/32

Open the official WireGuard app on your phone:
  1. Tap +
  2. Choose Create from QR code / Scan from QR code
  3. Scan this QR code
  4. Toggle the tunnel ON

[terminal QR code containing WireGuard client config]

Waiting for Alice iPhone to connect...
```

Success:

```text
Alice iPhone connected.

WireGuard:
  interface: glwg0
  last handshake: 4 seconds ago
  tunnel IP: 10.94.23.2
  transfer: 1.2 KiB received, 2.4 KiB sent

Open GridLens:
  http://10.94.23.1:8787

[terminal QR code for UI URL or pairing URL]
```

Timeout/no handshake:

```text
No handshake from Alice iPhone yet.

Server status:
  GridLens Hub: running
  WireGuard interface glwg0: active
  UDP 51871: listening
  Phone peer: no handshake

Likely causes:
  - WireGuard tunnel is not toggled ON in the phone app
  - phone scanned the wrong QR
  - endpoint IP/hostname is wrong
  - router port forward is missing
  - firewall blocks UDP 51871
  - ISP uses CGNAT

Run:
  gridlens doctor wireguard
```

### 8.5 UI access failure

There are two cases.

Case A: web UI never loads on the phone. The browser cannot diagnose this because it cannot reach the Hub. The CLI must diagnose using `gridlens doctor wireguard`.

Case B: web UI loaded once, then fails or is degraded. The UI can call Hub status APIs and show:

```text
WireGuard: active
Phone peer: last handshake 35 seconds ago
GridLens Hub: healthy
UI websocket: reconnecting
```

or:

```text
WireGuard peer is stale.
Last handshake: 17 minutes ago.
Open the WireGuard app and confirm the GridLens tunnel is ON.
```

---

## 9. WireGuard model details

### 9.1 Server config example

```ini
# Managed by GridLens. Do not edit manually while GridLens is running.
# Owner marker: gridlens
# Interface: glwg0

[Interface]
PrivateKey = <server_private_key>
Address = 10.94.23.1/24
ListenPort = 51871

# Peer: Alice iPhone
# PeerID: peer_abc123
[Peer]
PublicKey = <phone_public_key>
AllowedIPs = 10.94.23.2/32
```

### 9.2 Phone config example

```ini
[Interface]
PrivateKey = <phone_private_key>
Address = 10.94.23.2/32

[Peer]
PublicKey = <server_public_key>
Endpoint = <public-ip-or-hostname>:51871
AllowedIPs = 10.94.23.1/32
PersistentKeepalive = 25
```

### 9.3 Key handling

Server:

- Generate once.
- Store private key at `/etc/gridlens/wireguard/server.key` mode `0600`.
- Store public key separately or derive as needed.

Phone peer:

- Generate a new client keypair per peer.
- Add only the client public key to server config and peer metadata.
- Render client private key in QR code at pairing time.
- Do not persist client private key by default.
- If user loses QR/config, instruct them to revoke and re-add or rotate the peer.

Rationale: long-term storage of phone private keys makes the Hub a credential vault for every remote device. Avoid that by default.

### 9.4 Peer metadata example

```json
{
  "id": "peer_01hxyz...",
  "name": "Alice iPhone",
  "publicKey": "...",
  "assignedIP": "10.94.23.2",
  "createdAt": "2026-05-07T12:00:00Z",
  "revokedAt": null,
  "lastHandshakeAt": null,
  "notes": "created by gridlens phone add"
}
```

### 9.5 IP allocation

Use a private RFC1918 `/24` per Hub by default. Pick a deterministic-but-random-ish subnet at first setup to reduce collisions, for example:

```text
10.<hash_byte_1>.<hash_byte_2>.0/24
```

Avoid common subnets if possible:

```text
10.0.0.0/24
10.1.1.0/24
10.8.0.0/24
10.10.10.0/24
192.168.0.0/24
192.168.1.0/24
172.16.0.0/24
```

Default Hub IP:

```text
10.x.y.1
```

Peers:

```text
10.x.y.2 through 10.x.y.254
```

Reject duplicate peer IPs.

---

## 10. Dependency detection and installation

### 10.1 Required commands

Minimum Linux Hub dependencies:

```text
wg
wg-quick
ip
ss
systemctl, when installing as systemd service
```

Likely packages:

```text
wireguard-tools
iproute2
systemd
```

### 10.2 Supported package managers, in order

Initial implementation should support:

```text
apt-get: Ubuntu/Debian
```

Future:

```text
dnf/yum: Fedora/RHEL/Rocky/Alma
pacman: Arch
zypper: openSUSE
apk: Alpine, only if systemd service assumptions are disabled
```

### 10.3 Installation rules

- Never install silently.
- Show exact packages and command.
- Use targeted installs only.
- Avoid broad upgrades.
- If package manager locks are active, stop and explain.
- If existing WireGuard interfaces are active, default dependency update prompts to **No**.
- Log the commands that were run.

### 10.4 Dependency API shape

Implement dependency logic as a package that returns structured data, not strings only:

```go
type DependencyStatus struct {
    Command string
    Found bool
    Path string
    Version string
    Required bool
    PackageHint string
}

type PackageManager struct {
    Name string
    InstallCommand []string
    UpdateMetadataCommand []string
    CanCheckUpdates bool
}
```

This makes CLI and web diagnostics easier.

---

## 11. Systemd/service behavior

### 11.1 Services

Use two services:

```text
wg-quick@glwg0.service
GridLens-owned WireGuard interface, managed by wg-quick.

gridlens-hub.service
GridLens Hub API/UI service.
```

### 11.2 Do not restart unrelated services

Acceptable:

```bash
systemctl restart wg-quick@glwg0
systemctl restart gridlens-hub
```

Forbidden:

```bash
systemctl restart wg-quick@wg0
systemctl restart wg-quick.target
systemctl restart networking
systemctl restart NetworkManager
```

### 11.3 Unit behavior

`gridlens-hub.service` should:

- run as a dedicated `gridlens` user when practical;
- have access to `/etc/gridlens`, `/var/lib/gridlens`, `/var/log/gridlens`;
- bind to `127.0.0.1:8787` and the WG IP;
- not require root unless necessary;
- log to journald and optionally `/var/log/gridlens/hub.log`.

---

## 12. Firewall behavior

MVP should not automatically change firewall rules unless the user explicitly agrees.

Detection:

- detect UFW active/inactive;
- detect firewalld active/inactive;
- detect nftables/iptables presence;
- detect whether UDP listen port appears open locally;
- detect whether TCP 8787 is reachable on WG interface locally.

Suggested prompts:

```text
UFW is active and may block UDP 51871.
GridLens can add a narrow rule:
  sudo ufw allow 51871/udp comment 'gridlens glwg0'
Add this rule? [y/N/show command]
```

Default should be **No** until the user sees the reason.

No LAN-wide rule should be added. Do not expose TCP 8787 to all interfaces.

---

## 13. Router and CGNAT diagnostics

GridLens cannot guarantee remote inbound UDP without network support. It should provide a router instruction block:

```text
Router port-forward needed:
  Protocol: UDP
  External port: 51871
  Internal IP: 192.168.1.50
  Internal port: 51871
```

Diagnostics should collect:

- default route interface;
- local LAN IP;
- candidate IPv4/IPv6 endpoints;
- UDP listen port;
- whether WG interface is up;
- whether peer has handshaken;
- whether bytes are moving;
- whether public endpoint differs from local WAN candidate, when external IP check is enabled.

Do not depend on a third-party public IP service silently. If implemented, ask first:

```text
GridLens can check your public IP using a public web service.
This is optional and sends one request outside your network.
Check now? [y/N]
```

CGNAT likely diagnosis:

```text
Your router WAN address appears to be private/reserved, but the public IP check shows a different address.
This often means CGNAT. Direct inbound WireGuard may not work without ISP changes, IPv6, router support, or a future relay/VPS mode.
```

---

## 14. Hub web UI requirements

### 14.1 MVP pages

Initial web UI can be simple, but must be useful:

```text
/                 overview
/health           human-readable health
/api/healthz      machine health
/api/wireguard/status
/api/doctor/wireguard
/api/peers
```

Overview should show:

- GridLens Hub running status;
- WireGuard interface name/IP/status;
- connected peers;
- last handshake per peer;
- setup checklist;
- clear next step if phone is not connected.

### 14.2 Binding

Default bind addresses:

```text
127.0.0.1:8787
<gridlens_wg_ip>:8787
```

Do not bind to `0.0.0.0` by default.

### 14.3 Authentication/pairing

For MVP, WireGuard is the primary access control. Add a simple pairing/session layer if feasible:

- `gridlens ui pair` or successful `phone add` creates a short-lived pairing URL.
- Pairing token should be random, one-time-use, and expire within 10 minutes.
- The UI stores a session token or cookie.
- Session revocation should be tied to peer revocation when possible.

Do not block the first prototype on sophisticated auth, but do not expose UI beyond loopback/WG.

### 14.4 UI diagnostics

When loaded through the tunnel, the UI should show:

```text
WireGuard: active/inactive
Peer: connected/stale/no handshake
Last handshake: relative and absolute time
Hub: healthy/unhealthy
WebSocket/live update: connected/reconnecting
```

Remember: if the browser cannot reach the Hub at all, the browser cannot self-diagnose. The CLI doctor is required.

---

## 15. Doctor command requirements

`gridlens doctor wireguard` should produce structured checks and human-readable output.

### 15.1 Checks

Minimum checks:

```text
OS supported
running as root or sudo available when needed
commands present: wg, wg-quick, ip, ss
existing WG interfaces detected
GridLens interface exists
GridLens ownership marker valid
server private key exists and permissions are safe
wg-quick config exists and permissions are safe
interface is up/down
UDP listen port active/inactive
Hub process running/not running
Hub listening on 127.0.0.1:8787
Hub listening on WG IP:8787
firewall active/inactive/unknown
peer count
per-peer last handshake
per-peer transfer bytes
endpoint configured for phone configs
router port forward likely needed
CGNAT suspected/not checked
```

### 15.2 Output style

Example:

```text
GridLens WireGuard Doctor

PASS  OS supported: Ubuntu 24.04
PASS  WireGuard tools installed: wg, wg-quick
WARN  Existing WireGuard interfaces detected: wg0, corpvpn
PASS  GridLens interface name: glwg0
PASS  GridLens owns glwg0
PASS  glwg0 is active
PASS  UDP 51871 is listening
FAIL  No handshake from Alice iPhone
PASS  GridLens Hub is running
PASS  Hub is reachable at http://10.94.23.1:8787 from the Hub
WARN  Router port forward not verified

Likely next steps:
  1. On phone, open WireGuard and toggle the GridLens tunnel ON.
  2. Confirm router forwards UDP 51871 to 192.168.1.50.
  3. Run this again after trying.
```

### 15.3 Machine-readable output

Support JSON:

```bash
gridlens doctor wireguard --json
```

This enables future UI and issue templates.

---

## 16. Testing strategy

### 16.1 Unit tests

Required early tests:

- WireGuard config render preserves comments/markers.
- Peer addition creates exactly one `[Peer]` block.
- Peer revocation removes only the selected peer.
- IP allocator avoids duplicates and reserved addresses.
- Existing `wg0` detection never results in modification.
- `glwg0` ownership mismatch refuses modification.
- Dependency detector handles missing/found commands.
- Doctor maps raw command outputs into statuses.
- Phone config uses `AllowedIPs = <hub_ip>/32`, not `0.0.0.0/0`.
- Client private key is not persisted in peer metadata by default.

### 16.2 Integration tests

Optional/gated:

- Linux network namespace test for WireGuard interface creation.
- systemd dry-run unit rendering.
- UFW command suggestion rendering.

Integration tests should be skipped unless explicitly enabled:

```bash
GRIDLENS_INTEGRATION=1 go test ./...
```

### 16.3 Safety tests

Add tests that fail if code tries to manage unrelated interfaces:

```text
Given active interfaces: wg0, corpvpn
When GridLens setup runs
Then no command may contain wg0 or corpvpn except read-only status commands
```

This is a core project safety invariant.

---

## 17. Acceptance criteria for the first usable MVP

A PR or release is not “usable” until these are true:

1. On a fresh Ubuntu/Debian machine with no WireGuard, GridLens detects missing dependencies and prompts before installing.
2. On a machine with existing `wg0` active, GridLens creates only `glwg0` and does not restart or modify `wg0`.
3. `gridlens setup wireguard` creates server keys, config, and service with safe permissions.
4. `gridlens phone add "Test Phone"` creates a dedicated peer and renders a WireGuard QR code in the terminal.
5. Phone config uses hub-only routing: `AllowedIPs = <hub_ip>/32`.
6. CLI waits for handshake and shows success/failure diagnostics.
7. `gridlens phone revoke` removes only the selected peer.
8. `gridlens doctor wireguard` produces useful output for missing dependency, inactive tunnel, no handshake, and Hub not running.
9. Hub UI is reachable over the WireGuard IP after handshake.
10. Hub UI is not exposed on `0.0.0.0` by default.
11. Tests cover the “do not break existing WireGuard” invariants.
12. README explains the network reality: direct WireGuard may require router port forwarding, public IP, reachable IPv6, or future relay/VPS mode.

---

## 18. Suggested implementation phases for Codex

### Phase 0 — repository scaffold

Goal: create a clean repo that can build and test.

Tasks:

- Add `go.mod`.
- Add CLI skeleton.
- Add README with project goals and current MVP scope.
- Add root `AGENTS.md`.
- Add docs folder with this handoff.
- Add GitHub Actions for `go test ./...`.
- Add placeholder `start.sh` that prints current status and does not perform privileged changes yet.

Acceptance:

```bash
go test ./...
go run ./cmd/gridlens version
```

### Phase 1 — dependency and system discovery

Goal: safe read-only detection.

Tasks:

- Detect OS/distro/package manager.
- Detect `wg`, `wg-quick`, `ip`, `ss`, `systemctl`.
- Detect existing WireGuard interfaces with read-only commands.
- Detect if `glwg0` exists and whether GridLens owns it.
- Add `gridlens deps check` or include in `gridlens setup wireguard --dry-run`.
- Add tests with fake command runner.

Acceptance:

```bash
gridlens setup wireguard --dry-run
gridlens doctor wireguard --json
```

No privileged modifications.

### Phase 2 — WireGuard config and peer management

Goal: generate safe config without yet requiring live interface.

Tasks:

- Generate server keypair.
- Generate client keypairs.
- Render `glwg0.conf`.
- Add/revoke/list peer metadata.
- Terminal QR code for client config.
- Do not persist client private keys by default.
- Tests for config render and revocation.

Acceptance:

```bash
gridlens setup wireguard --dry-run
gridlens phone add "Alice iPhone" --dry-run
gridlens phone list
```

### Phase 3 — live WireGuard service

Goal: create/start `glwg0` safely.

Tasks:

- Write config atomically with safe permissions.
- Start/restart only `wg-quick@glwg0`.
- Check status via `wg show glwg0`.
- Implement handshake watcher.
- Implement disable/enable.
- Never touch unrelated interfaces.

Acceptance:

```bash
gridlens setup wireguard
gridlens wireguard status
gridlens doctor wireguard
```

### Phase 4 — Hub UI over tunnel

Goal: phone can open a minimal web UI.

Tasks:

- Add Hub HTTP server.
- Bind to loopback and WG IP only.
- Show status page.
- Add `/api/healthz` and `/api/wireguard/status`.
- Add second QR code/URL for UI after handshake.

Acceptance:

```bash
gridlens hub start
gridlens phone add "Alice iPhone"
# after phone handshake, open http://<hub_wg_ip>:8787
```

### Phase 5 — doctor polish and network guidance

Goal: make failures understandable.

Tasks:

- Add clear diagnosis for no handshake.
- Add router port-forward instructions.
- Add firewall warning/suggestion.
- Add optional public IP check with explicit permission.
- Add JSON doctor output.

Acceptance:

- User sees likely next step, not raw WireGuard confusion.

### Phase 6 — Nosana host monitoring MVP

Goal: basic local host status after remote access is stable.

Tasks:

- Detect Nosana process/container.
- Detect GPU status from `nvidia-smi` if available.
- Show host health in Hub UI.
- Store snapshots in SQLite.
- Keep on-chain/API enrichment optional and deferred.

---

## 19. Data model starter

SQLite tables, approximate:

```sql
CREATE TABLE peers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  public_key TEXT NOT NULL UNIQUE,
  assigned_ip TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  revoked_at TEXT,
  notes TEXT
);

CREATE TABLE events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TEXT NOT NULL,
  level TEXT NOT NULL,
  source TEXT NOT NULL,
  message TEXT NOT NULL,
  details_json TEXT
);

CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE host_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TEXT NOT NULL,
  host_id TEXT NOT NULL,
  payload_json TEXT NOT NULL
);
```

WireGuard config can be derived from config + peers table, but keep a simple file-based metadata store acceptable for Phase 1-3. Do not over-engineer before live tunnel works.

---

## 20. Important future requirements, not MVP blockers

Later modes:

- relay mode for no-router remote access;
- Tor onion service mode for no-account private remote access;
- public HTTPS via Caddy;
- private relay/VPS customer mode;
- GitHub Pages static mirror only as optional convenience;
- Nosana Kit/Solana Kit enrichment;
- optional Nosana API connector with user-provided auth;
- multi-host agent-to-hub telemetry;
- alerts and notification integrations.

Do not accidentally design the WireGuard MVP so these modes become impossible. Keep the Hub/transport boundary clean.

---

## 21. Documentation requirements

Update docs whenever behavior changes:

```text
README.md                         user-facing quick start and current limitations
docs/WIREGUARD_PRIVATE_REMOTE_MODE.md detailed private remote mode behavior
docs/SECURITY.md                  threat model and key handling
docs/TROUBLESHOOTING.md           doctor outputs and fixes
docs/CODEX_HANDOFF.md             implementation decisions and phase plan
```

README should be honest that remote WireGuard requires a reachable UDP endpoint or a future relay/tunnel path.

---

## 22. PR checklist

Every PR touching WireGuard/system setup must answer:

- Did this change modify only GridLens-owned resources?
- Can this change affect existing `wg0` or other WireGuard tunnels?
- Does the phone config still use hub-only `AllowedIPs`?
- Does this change run privileged commands? If yes, are they prompted/logged?
- Are private keys handled safely?
- Are tests added/updated?
- Does `gridlens doctor wireguard` explain the new failure mode?
- Is documentation updated?

---

## 23. Forbidden shortcuts

Do not implement these shortcuts, even if they make a demo faster:

```text
- edit /etc/wireguard/wg0.conf
- use wg0 as GridLens interface
- run apt upgrade
- silently install packages
- bind Hub UI to 0.0.0.0 by default
- expose unauthenticated UI to the LAN/public internet
- persist phone private keys by default
- set phone AllowedIPs to 0.0.0.0/0
- enable NAT/IP forwarding by default
- restart all WireGuard services
- require Cloudflare/GitHub Pages/Tailscale/API keys for MVP
- hide router/CGNAT limitations
```

---

## 24. Suggested initial Codex prompt

Use this when starting a fresh Codex task:

```text
We are starting a fresh repo for GridLens for Nosana Hosts. Read AGENTS.md and docs/CODEX_HANDOFF.md first. Implement Phase 0 and Phase 1 only: Go CLI scaffold, docs, dependency/system discovery, fake command runner tests, and dry-run WireGuard doctor output. Do not perform privileged operations except in code paths guarded by explicit prompts or dry-run. Do not create or modify real WireGuard interfaces in tests. The most important invariant is: never modify existing WireGuard setups such as wg0; GridLens may only own glwg0 or another explicitly marked GridLens interface. Open a PR with tests and README updates.
```

Then subsequent prompts can use the phase plan above.

---

## 25. Open owner decisions

These have safe defaults. Codex should use the defaults unless the owner says otherwise.

| Decision | Safe default |
|---|---|
| Product name | GridLens |
| Repo name | `nosana-gridlens` |
| Interface name | `glwg0` |
| Hub port | `8787` |
| WireGuard UDP port | first free port in `51871-51999` |
| Routing | hub-only, `AllowedIPs = <hub_ip>/32` |
| Server storage | `/etc/gridlens`, `/var/lib/gridlens`, `/var/log/gridlens` |
| Client private key storage | do not persist by default |
| First supported OS | Ubuntu/Debian with systemd |
| First implementation language | Go |
| First UI | embedded minimal static UI |

---

## 26. How Codex should report uncertainty

Codex should not invent network behavior or overpromise remote access. If direct WireGuard cannot work because no UDP path exists, the correct answer is a diagnostic with options, not a broken workaround.

When blocked, Codex should add this section to the PR summary:

```text
## Blocked / needs owner decision

- <specific issue>
- Safe default used: <default>
- Alternative options: <option A>, <option B>
```

---

## 27. Source notes for Codex behavior

OpenAI Codex docs state that Codex can read, edit, and run code in cloud tasks, and that connecting a GitHub account lets it work with repositories and create pull requests. Official Codex docs also state that Codex reads `AGENTS.md` files before doing work, so this repo should include one at the root and keep it concise enough to remain within Codex instruction limits.

Relevant official docs:

- `https://developers.openai.com/codex/cloud`
- `https://developers.openai.com/codex/guides/agents-md`
- `https://developers.openai.com/codex/integrations/github`

---

## 28. Final implementation principle

The demo should be slower if necessary, but it must be safe.

A monitoring tool for Nosana operators will lose trust immediately if it breaks an existing VPN, exposes a private UI, or hides a networking problem. The winning UX is not magic; it is clear, guided, reversible setup with excellent diagnostics.
