# Security

GridLens is built around a conservative default: do not expose private host
monitoring surfaces and do not alter existing VPNs.

## WireGuard

- GridLens never uses `wg0`.
- GridLens defaults to `glwg0`.
- Existing WireGuard interfaces are read-only unless GridLens can prove
  ownership.
- GridLens does not enable NAT or IP forwarding by default.
- GridLens does not set phone `AllowedIPs = 0.0.0.0/0` by default.
- GridLens does not bind the future Hub UI to `0.0.0.0` by default.

## Keys

- Nosana and Solana private keys must never be touched, copied, uploaded,
  printed, or persisted by GridLens.
- Future WireGuard server private keys must be stored with `0600` permissions.
- Future phone/client private keys must not be persisted by default.

## Privileged Operations

The current implementation runs no privileged operations. Future privileged
operations must show the exact command and require explicit user confirmation.
