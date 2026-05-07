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

Current monitor data:

- configured PCs;
- local Docker/Podman availability;
- configured SSH Docker/Podman availability;
- matching Nosana containers by exact name or configured patterns;
- Podman nested inside Docker containers;
- LAN candidates from selected TCP ports.

Current limits:

- no login/auth layer yet because the default bind is loopback-only;
- no remote WireGuard tunnel UI binding yet;
- no GPU, memory, disk, or log telemetry yet;
- no web form for adding PCs yet.
