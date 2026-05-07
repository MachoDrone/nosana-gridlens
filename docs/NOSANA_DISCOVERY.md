# Nosana Discovery

GridLens should discover real Nosana host data. It should not invent placeholder
nodes or mock fleet records.

## Commands

```bash
gridlens pc scan [--cidr 192.168.0.0/24] [--ports 22,2375,2376]
gridlens pc add NAME --address IP [--ssh user@host] [--container NAME] [--pattern GLOB]
gridlens pc list
gridlens pc remove NAME
gridlens nosana detect [--json]
```

## Container Runtimes

Discovery currently checks:

- local Docker;
- local Podman;
- Docker over configured SSH targets;
- Podman over configured SSH targets;
- Podman nested inside Docker containers.

All runtime checks are read-only status commands such as `docker ps`,
`podman ps`, and `docker exec <container> podman ps` for nested Podman
discovery.

## Custom Names

Nosana containers are often named `nosana-node`, but operators can customize
names and can run multiple hosts on one PC. GridLens therefore supports exact
container names and glob patterns:

```bash
gridlens pc add gpu-box-1 \
  --address 192.168.0.167 \
  --ssh grid@192.168.0.167 \
  --container nosana-a \
  --container nosana-b \
  --pattern "nosana-*"
```

User-level config is stored at:

```text
~/.config/gridlens/config.json
```

This file contains host discovery hints only. It must not contain private keys.

## Network Scan

`pc scan` is part of the app. It searches for candidate PCs by opening TCP
connections to selected ports. It does not attempt password guessing, SSH login,
package installation, firewall changes, or privileged operations.

Default scan behavior is intentionally bounded:

- local non-loopback private IPv4 CIDRs only;
- `/24` or narrower auto-detected networks only;
- default ports: `22`, `2375`, `2376`;
- default host cap: `1024`.
