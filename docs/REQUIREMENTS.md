# Requirements

## Current Phase

Phase 0 and Phase 1 provide a buildable Go CLI with read-only system discovery.

Required for development:

- Go 1.22 or newer
- Git

Required on a future Ubuntu/Debian Hub for WireGuard mode:

- `wg`
- `wg-quick`
- `ip`
- `ss`
- `systemctl` when installing services

GridLens detects these commands before suggesting any install command. It does
not install or update packages silently.

## First Supported Platform

Ubuntu/Debian with systemd is the first supported platform. Other package
managers are detected for diagnostics, but install behavior is not implemented
in this phase.
