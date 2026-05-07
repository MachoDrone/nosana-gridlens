# WireGuard Private Remote Mode

WireGuard Private Remote Mode gives a phone or PC private access to the local
GridLens Hub through a GridLens-owned tunnel.

## Defaults

- Interface: `glwg0`
- Hub UI port: `8787`
- WireGuard UDP port range: `51871-51999`
- Routing: hub-only
- Phone `AllowedIPs`: `<hub_wg_ip>/32`

## Ownership

GridLens may manage only resources that it created and marked as GridLens-owned.
If `glwg0` exists without a GridLens ownership marker, GridLens must refuse to
modify it. Existing interfaces such as `wg0` are read-only.

## Current Phase

The current implementation performs dry-run discovery only:

```bash
gridlens setup wireguard --dry-run
gridlens doctor wireguard --json
```

No WireGuard config is written, no service is restarted, and no privileged
command is run.
