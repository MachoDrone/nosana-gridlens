# Cluster Protocol Direction

GridLens may run on any Nosana host PC for redundancy. Customers may already
operate 100 Nosana hosts and future users may operate 200 or more. The best
steady-state protocol is not SSH polling with saved passwords. SSH is useful
for discovery and bootstrap, but the long-term monitoring plane should be a
small GridLens agent on each PC.

Terminology:

- **PC**: a physical or virtual machine that runs one or more container
  runtimes.
- **Nosana host**: the actual `nosana-node` container or an operator-chosen
  custom container name.
- A PC count is not the same as a Nosana host count. One PC may run multiple
  Nosana hosts.
- Runtime wrapper containers, such as Docker containers that exist only to host
  nested Podman, are not Nosana hosts when nested `nosana-node` containers are
  present.

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
- A single Hub should comfortably ingest 100-200 host snapshots without relying
  on SSH fan-out.

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
- Gossip can be added later for Hub/Agent membership discovery, but metrics
  should still be transported as authenticated snapshots.

## Why Not Saved SSH Password Polling

Saved SSH passwords make the Hub a credential vault and create a larger blast
radius if the monitor is compromised. Password polling also scales poorly when
multiple PCs run different credentials. GridLens should support passwords only
as a bootstrap convenience, preferably session-only, and move persistent
monitoring to mTLS agent identity.

## Where Gossip Fits

Gossip is useful when nodes need decentralized membership awareness. It is not
the primary security layer and it is not required for the first 100-200 host
monitoring path.

Use gossip later for:

- sharing which Hubs are alive;
- sharing which Agents were recently seen;
- reducing manual Hub endpoint config in larger fleets.

Do not use gossip as the only source of truth for command authorization or
credentials. Agent-to-Hub traffic should still use mTLS identity and explicit
authorization.
