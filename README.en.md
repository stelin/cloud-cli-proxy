<div align="center">

# Cloud CLI Proxy

**One command. One cloud machine. All traffic through your exit IP.**

[![CI](https://github.com/ZaneL1u/cloud-cli-proxy/actions/workflows/build-images.yml/badge.svg)](https://github.com/ZaneL1u/cloud-cli-proxy/actions/workflows/build-images.yml)
[![Release](https://img.shields.io/github/v/release/ZaneL1u/cloud-cli-proxy)](https://github.com/ZaneL1u/cloud-cli-proxy/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[Chinese](README.md) | [Documentation](https://zanel1u.github.io/cloud-cli-proxy/en/)

</div>

---

Cloud CLI Proxy is a single-host containerized SSH cloud platform. Users get a dedicated Docker container with a single `curl` command. All egress traffic is routed through WireGuard full-tunnel to designated exit IPs, preventing any DNS/WebRTC or other direct leaks.

## Key Features

- **One-command access** -- `curl | bash` to authenticate, create container, and establish SSH session
- **Full-tunnel egress** -- WireGuard + Linux netns with nftables default-deny policy, zero leaks
- **Per-user isolation** -- Docker containers with pre-installed Claude Code, KasmVNC desktop, and Chromium
- **Flexible egress IP management** -- Multi-IP pool, per-user binding, connectivity testing
- **Automatic expiration** -- Auto-stop containers and block login on expiry
- **Admin dashboard** -- React SPA covering users, hosts, egress IPs, events, and stats
- **Multi-arch CI/CD** -- GitHub Actions builds `linux/amd64` + `linux/arm64` images

## Quick Start

```bash
# 1. Clone
git clone https://github.com/ZaneL1u/cloud-cli-proxy.git
cd cloud-cli-proxy

# 2. Generate config (interactive, auto-generates passwords and secrets)
bash deploy/scripts/setup-env.sh

# 3. Start
docker compose up -d --build

# 4. Verify
curl http://127.0.0.1:8080/healthz
# {"status":"ok"}
```

Once running, admins create users and egress IPs via API, then share this with users:

```bash
curl -sSf http://YOUR_HOST:8080/v1/bootstrap/script | bash
```

## Architecture

```
End User ──curl──> Control Plane (Go API) ──Docker──> User Container (SSH + VNC + Claude)
                        │                                    │
                   PostgreSQL                          WireGuard Tunnel
                                                            │
                                                      Designated Exit IP
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go, net/http, pgx v5 |
| Frontend | React 19, TypeScript, Vite, Tailwind CSS |
| Database | PostgreSQL 18 |
| Containers | Docker Engine 28, Ubuntu 24.04 |
| Networking | WireGuard + Linux netns, nftables, sing-box |
| Desktop | KasmVNC + Fluxbox + Chromium |

## Documentation

Full documentation is hosted on [GitHub Pages](https://zanel1u.github.io/cloud-cli-proxy/en/), including:

- [Quick Start](https://zanel1u.github.io/cloud-cli-proxy/en/guide/quickstart) -- Deploy and first use
- [Deployment Guide](https://zanel1u.github.io/cloud-cli-proxy/en/guide/deployment) -- Detailed systemd native deployment
- [Configuration](https://zanel1u.github.io/cloud-cli-proxy/en/guide/configuration) -- Environment variables and WireGuard setup
- [Architecture](https://zanel1u.github.io/cloud-cli-proxy/en/guide/architecture) -- System design and project structure
- [API Reference](https://zanel1u.github.io/cloud-cli-proxy/en/reference/api) -- Full Admin API docs
- [FAQ & Recovery](https://zanel1u.github.io/cloud-cli-proxy/en/reference/faq) -- Troubleshooting and disaster recovery

## Development

```bash
make setup    # Install deps, copy .env.example
make db       # Start PostgreSQL
make dev      # Backend + frontend with hot reload
make test     # Run all tests
```

See `make help` for all available commands.

## Contributing

Issues and Pull Requests are welcome. Please use [Conventional Commits](https://www.conventionalcommits.org/) format.

## License

[MIT](LICENSE)
