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
by name or IP address, and the browser persists that sort choice. Containers
within a PC are ordered with matched Nosana host containers first, then all
remaining containers alphabetically.

LAN scan candidates that are not configured or do not currently expose a
Nosana host are dimmed with a short reason, such as missing credentials or no
Nosana host discovered. The local Hub PC is collapsed when it is not itself a
Nosana host.

Current limits:

- no login/auth layer yet because the default bind is loopback-only;
- no remote WireGuard tunnel UI binding yet;
- no GPU, memory, disk, or log telemetry yet;
- no password-based persistent polling yet;
- no GridLens agent protocol yet.
