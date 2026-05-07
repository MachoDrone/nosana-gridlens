# Cluster Protocol Direction

GridLens may run on any Nosana host PC for redundancy. The best steady-state
protocol is not SSH polling with saved passwords. SSH is useful for discovery
and bootstrap, but the long-term monitoring plane should be a small GridLens
agent on each PC.

## Recommended Model

```text
GridLens Hub candidates
  - run local monitor UI
  - store recent signed host snapshots
  - can become active monitor if another Hub is down

GridLens Agent on each Nosana PC
  - reads local Docker/Podman/Nosana/GPU stats
  - never reads Nosana/Solana private keys
  - publishes signed snapshots
  - accepts config only from trusted GridLens Hub identities
```

Transport:

- WireGuard private network first.
- Mutual TLS between GridLens Hubs and Agents.
- Agent pushes status snapshots every few seconds.
- Hubs can also request an immediate refresh.
- Snapshots are signed or authenticated by mTLS identity.

Data format:

- JSON for the first implementation.
- Later, protobuf/gRPC is a reasonable upgrade when the API stabilizes.

Discovery/bootstrap:

- Use LAN scan to find candidate PCs.
- Use SSH keys or a one-time password session to install/start an agent.
- Do not persist plaintext passwords in `~/.config/gridlens/config.json`.

Redundancy:

- Each Hub keeps a recent cache of agent snapshots.
- Multiple Hubs can subscribe to the same agents.
- Agents should know a list of Hub endpoints and publish to any reachable Hub.
- A future leader election or simple priority list can decide which Hub sends
  commands or configuration changes.

## Why Not Saved SSH Password Polling

Saved SSH passwords make the Hub a credential vault and create a larger blast
radius if the monitor is compromised. Password polling also scales poorly when
multiple PCs run different credentials. GridLens should support passwords only
as a bootstrap convenience, preferably session-only, and move persistent
monitoring to mTLS agent identity.
