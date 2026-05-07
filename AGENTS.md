# AGENTS.md — GridLens repository instructions

## Project mission

GridLens is a self-hostable monitoring console for Nosana host operators. The first implementation target is **Private Remote Mode with WireGuard**: a secure way for a user to view the local GridLens Hub from a phone or PC without requiring Cloudflare, GitHub Pages, Tailscale, Nosana API keys, Solana RPC keys, hosted databases, or cloud workers.

## Read first

Before implementation, read `docs/CODEX_HANDOFF.md`. If present, also read `docs/WIREGUARD_PRIVATE_REMOTE_MODE.md`, `docs/REQUIREMENTS.md`, `docs/SECURITY.md`, and the README.

## Non-negotiable safety rules

- Never break or modify existing WireGuard setups.
- Do not use `wg0` for GridLens. Use `glwg0` by default.
- Do not edit `/etc/wireguard/wg0.conf` or restart unrelated WireGuard services.
- GridLens may only manage resources it created and marked as GridLens-owned.
- Do not set phone `AllowedIPs = 0.0.0.0/0` by default.
- Use hub-only routing for MVP: phone peers may reach only the GridLens Hub IP, for example `10.x.y.1/32`.
- Do not enable NAT, IP forwarding, LAN-wide routing, or global DNS changes by default.
- Do not silently install packages or run broad upgrades such as `apt upgrade`.
- Do not bind the Hub UI to `0.0.0.0` by default.
- Do not persist phone/client WireGuard private keys by default.
- Never touch, copy, upload, print, or persist Nosana/Solana private keys.

## Implementation preferences

- Prefer Go for the initial CLI/Hub so the runtime can become a single binary.
- Keep system actions behind clear prompts and dry-run paths.
- Build command execution behind an interface so tests can use fake command runners.
- Prefer structured status objects over parsing display strings.
- Add tests for safety invariants, especially “existing wg0 is never modified.”
- The first supported platform is Ubuntu/Debian with systemd; keep future package-manager support modular.

## MVP commands

Target commands:

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
```

## Testing expectations

Run before opening a PR:

```bash
go test ./...
```

When modifying WireGuard/system setup code, include tests that prove:

- unrelated interfaces such as `wg0` and `corpvpn` are read-only;
- GridLens config uses `glwg0` or another GridLens-owned interface only;
- peer configs use hub-only `AllowedIPs`;
- selected peer revocation does not remove other peers;
- client private keys are not stored in peer metadata by default.

## Documentation expectations

Update docs when behavior changes. README must be honest that direct WireGuard remote access requires a reachable UDP endpoint, such as a public IP, router port-forward, reachable IPv6, or a future relay/VPS mode.

## PR summary requirements

Each PR should include:

- What changed.
- How it was tested.
- Any privileged operations added or changed.
- Whether existing WireGuard setups can be affected. The expected answer should normally be “No; only GridLens-owned resources are touched.”
- Any blocked decisions or safe defaults used.
