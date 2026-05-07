# GridLens

GridLens is a self-hostable monitoring console for Nosana host operators.

The first implementation target is Private Remote Mode with WireGuard: a local
GridLens Hub reachable from a phone or PC over a GridLens-owned WireGuard
tunnel.

Current status: Phase 0 and Phase 1 are implemented. The CLI can build, report
its version, discover host dependencies, detect existing WireGuard interfaces
with read-only commands, and produce a dry-run setup plan plus JSON doctor
output. It does not create interfaces, install packages, start services, or
modify system configuration yet.

## Safety Rules

GridLens is designed to avoid breaking existing WireGuard setups.

- GridLens never uses `wg0`.
- The default GridLens interface is `glwg0`.
- Existing tunnels such as `wg0` and `corpvpn` are read-only.
- `setup wireguard --dry-run` performs discovery only.
- Tests use fake command runners so test execution never creates or modifies
  WireGuard interfaces.
- Phone/client routing will be hub-only by default:
  `AllowedIPs = <hub_wg_ip>/32`.

## Build And Run

```bash
go test ./...
go run ./cmd/gridlens version
go run ./cmd/gridlens deps check
go run ./cmd/gridlens setup wireguard --dry-run
go run ./cmd/gridlens doctor wireguard --json
go run ./cmd/gridlens pc scan --json
go run ./cmd/gridlens nosana detect --json
go run ./cmd/gridlens hub start
```

## Remote Access Reality

Direct WireGuard access requires a reachable UDP path to the Hub. At least one
of these must be true:

- the Hub has a public IP and the selected UDP port is open;
- the router forwards the selected UDP port to the Hub;
- globally reachable IPv6 is available and the firewall allows the UDP port;
- a future relay, VPS, or tunnel mode is used.

GridLens must diagnose these conditions clearly instead of pretending remote
access can always work automatically.

## Nosana Data

GridLens uses real discovery, not mock fleet data.

GridLens separates PCs from Nosana hosts. A PC is a physical or virtual machine
that may run native Docker, native Podman, or Podman nested inside Docker. A
Nosana host is the actual `nosana-node` container or an operator-chosen custom
container name. One PC can run multiple Nosana hosts.

Current discovery commands:

```bash
gridlens hub start
gridlens pc scan
gridlens pc add nodebox --address 192.168.0.167 --ssh grid@192.168.0.167 --container custom-host-a
gridlens pc list
gridlens nosana detect
```

`hub start` serves the local monitor at `http://127.0.0.1:8787` by default.
It refreshes real Nosana discovery data through the Hub API and includes a LAN
candidate scan action. Use the Config button in the monitor to add individual
IPs, comma-separated IPs, ranges, or small CIDRs.

`pc scan` actively probes selected TCP ports on local `/24` networks or a CIDR
you provide. It defaults to ports `22`, `2375`, and `2376` so GridLens can find
SSH candidates and exposed Docker API candidates without requiring privileged
operations.

`nosana detect` inspects local Docker, local Podman, configured SSH PCs, and
Podman nested inside Docker containers using read-only status commands. Manual
PC/container config exists because Nosana host container names can be customized
and one PC may run multiple Nosana hosts.

Passwords entered through the monitor are not written to disk. For persistent
cross-PC collection, GridLens should prefer SSH keys now and a future
GridLens-agent protocol with mutual TLS.

The SSH discovery path uses bounded concurrency so larger fleets can be
inspected, but 100-200 host customers should ultimately use the planned
GridLens-agent snapshot protocol rather than long-term SSH polling.
