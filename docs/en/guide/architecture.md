# Architecture

## System Architecture

```
┌─────────────┐   curl    ┌────────────────┐    Docker     ┌──────────────────┐
│  End User   │ ────────> │  Control Plane │ ───────────>  │  User Container  │
│ (SSH Client)│ <──────── │  (Go API)      │              │  SSH + VNC + Claude│
└─────────────┘   SSH     └───────┬────────┘              └────────┬─────────┘
                                  │                                │
                           ┌──────┴───────┐               ┌───────┴────────┐
                           │  PostgreSQL   │               │  WireGuard /   │
                           │  (State store)│               │  sing-box tunnel│
                           └──────────────┘               └───────┬────────┘
                                                                  │
                                                          ┌───────┴────────┐
                                                          │  Designated    │
                                                          │  Exit IP       │
                                                          └────────────────┘
```

## Core Components

| Component | Responsibility |
|-----------|---------------|
| **Control Plane** | HTTP API, user auth, session management, expiry scanner, reconciler |
| **Host Agent** | Docker container lifecycle, WireGuard tunnels, nftables firewall, network namespaces |
| **User Container** | OpenSSH server, shell tools, Claude Code, KasmVNC desktop |
| **PostgreSQL** | Users, hosts, egress IP bindings, sessions, expiry, and audit events |

## Architecture Principles

- **Single-host first** -- No multi-node scheduling complexity in v1
- **Network enforcement first** -- All traffic must go through designated exits
- **Startup experience** -- Built on verifiable runtime correctness

## Key Boundaries

- Web/API layer does not hold excessive host privileges
- Docker and network namespace operations are centralized in host-agent
- User containers' default egress must be taken over by tunnel networking, no bypass allowed

## Project Structure

```
cloud-cli-proxy/
├── cmd/
│   ├── control-plane/          # Control plane API entrypoint
│   └── host-agent/             # Host agent entrypoint
├── internal/
│   ├── controlplane/           # HTTP routes, business logic, expiry scanner, reconciler
│   ├── agent/                  # Host-agent server
│   ├── network/                # WireGuard / nftables / sing-box config
│   ├── runtime/                # Docker container lifecycle
│   ├── sshproxy/               # SSH proxy (forwards to container port 22)
│   └── store/                  # Database migrations and queries (pgx)
├── web/admin/                  # React admin dashboard (TanStack Router)
├── deploy/
│   ├── docker/                 # 4 Dockerfiles
│   ├── compose/                # Dev Compose files
│   ├── bootstrap/              # User curl bootstrap script
│   ├── scripts/                # setup-env.sh, deploy.sh, backup.sh
│   └── systemd/                # systemd service units
├── docs/                       # VitePress docs site
├── docker-compose.yml          # Production Compose
├── Makefile                    # Dev command entrypoint
└── .github/workflows/          # CI/CD
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go, net/http stdlib, pgx v5 |
| Frontend | React 19, TypeScript, Vite, Tailwind CSS, TanStack Router/Query |
| Database | PostgreSQL 18 |
| Containers | Docker Engine 28, Ubuntu 24.04 user image |
| Networking | WireGuard + Linux netns, nftables, sing-box |
| Desktop | KasmVNC 1.4.0 + Fluxbox + Chromium |
