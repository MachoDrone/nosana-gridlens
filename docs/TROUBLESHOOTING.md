# Troubleshooting

Use the WireGuard doctor for structured diagnostics:

```bash
gridlens doctor wireguard
gridlens doctor wireguard --json
```

Current checks include:

- supported OS family;
- required command availability;
- package manager detection;
- existing WireGuard interfaces;
- `glwg0` ownership state;
- confirmation that no privileged operation was run.

Direct remote access over WireGuard may require router port forwarding, a
public IP, reachable IPv6, or a future relay/VPS mode.
