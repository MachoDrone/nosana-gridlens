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

GridLens should use real local Nosana host data, not mock fleet data. This
initial phase does not include the Hub UI or Nosana telemetry collectors yet;
those should be added as read-only live discovery and monitoring in a later
phase after the WireGuard safety foundation is in place.
