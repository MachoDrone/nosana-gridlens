# Hub Monitor

Start the local monitor:

```bash
gridlens hub start
```

Default bind:

```text
127.0.0.1:8787
```

The Hub does not bind to `0.0.0.0` by default. It serves:

```text
/                  monitor UI
/api/healthz       Hub health
/api/config        GridLens user config
/api/nosana        live Nosana discovery report
/api/pc/scan       bounded LAN candidate scan
```

The monitor uses the same real read-only discovery code as the CLI. It does not
invent nodes or display mock fleet data.

The UI has two local screens:

- `Monitor`, the PC-oriented operational view;
- `Fleet`, a dense sortable table with one row per discovered Nosana host
  container.

The Fleet screen intentionally follows the compact Nosana Fleet table layout:
black standalone surface, monospace text, vertical sortable headers, and the
same broad column shape. Columns that GridLens does not collect yet are rendered
as not-collected dashes instead of fabricated data.

Only the active screen polls. The Monitor screen is idle while Fleet is active,
Fleet is idle while Monitor is active, and all polling stops when the browser
tab is hidden.

Use the Config button to add monitored PCs from:

- individual IPs;
- comma-separated random IPs;
- short ranges such as `192.168.0.100-120`;
- full ranges such as `192.168.0.100-192.168.0.130`;
- small CIDRs.

The Config form can apply one username across a batch of IPs. Add separate
entries when each PC needs a unique login. Passwords entered in the form are not
persisted to disk.

Current monitor data:

- configured PCs;
- local Docker/Podman availability;
- configured SSH Docker/Podman availability;
- matching Nosana host containers by `nosana-node`, exact custom name, or
  configured patterns;
- Podman nested inside Docker containers;
- LAN candidates from selected TCP ports.

The monitor separates carrier PCs from Nosana hosts. A customer may have 6 PCs
and 8 Nosana hosts, or 80 PCs and 200 Nosana hosts. Runtime wrapper containers
are shown as containers but should not inflate the Nosana host count when nested
Nosana containers are found.

PC rows show both configured PC name and IP address. The operator can sort PCs
by name or IP address, and the browser persists that sort choice. When SSH is
configured, GridLens reads the remote hostname and uses it as the visible PC
name, while keeping the generated/configured name as metadata.

Containers within a PC are ordered with matched Nosana host containers first,
then all remaining containers alphabetically.

LAN scan candidates that are not configured or do not currently expose a
Nosana host are dimmed with a short reason, such as missing credentials or no
Nosana host discovered. Dimmed candidates are collapsed by default. The local
Hub PC is collapsed when it is not itself a Nosana host. If it is a Nosana host,
it is included in the normal sorted PC list.

The Hub PC name and local IP address are displayed under the update timestamp
as `Using PC <name> | IP <address>`.

The PC metric includes estimated GridLens network traffic for this Hub over the
last 60 seconds. The visible value is the rolling peak bandwidth estimate, and
the tooltip includes total bytes for the same window. SSH byte counts are
estimated because the encrypted SSH handshake and transport are owned by the
local `ssh` binary, not the Go process. The estimate includes a conservative
per-SSH-session overhead plus command payload, command output, and bounded LAN
port checks. Local Docker/Podman checks do not add network traffic.

If multiple Hub instances actively poll the same PCs, each Hub creates its own
SSH sessions and LAN scan traffic. Redundant instances should eventually use a
single active collector with peer sharing or leader election so monitoring
traffic does not multiply with every standby Hub.

Dark mode is the default monitor theme. The selected theme is stored in the
browser before the stylesheet loads so page refreshes do not flash the opposite
theme first.

Current limits:

- no login/auth layer yet because the default bind is loopback-only;
- no remote WireGuard tunnel UI binding yet;
- no GPU, memory, disk, or log telemetry yet;
- no password-based persistent polling yet;
- no GridLens agent protocol yet.
