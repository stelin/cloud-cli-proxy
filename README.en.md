<div align="center">

<img src="web/admin/public/favicon.svg" width="88" height="88" alt="Cloud CLI Proxy" />

# Cloud CLI Proxy

**One command. One cloud machine. All traffic through your exit IP.**

Out-of-the-box isolated cloud hosts for Claude Code and dev teams. Pre-installed AI coding tools, full-tunnel egress through designated IPs, zero leaks.

[![CI](https://github.com/ZaneL1u/cloud-cli-proxy/actions/workflows/build-images.yml/badge.svg)](https://github.com/ZaneL1u/cloud-cli-proxy/actions/workflows/build-images.yml)
[![Release](https://img.shields.io/github/v/release/ZaneL1u/cloud-cli-proxy)](https://github.com/ZaneL1u/cloud-cli-proxy/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[中文](README.md) | [Documentation](https://zanel1u.github.io/cloud-cli-proxy/en/)

**Go · React · PostgreSQL · Docker · WireGuard**

</div>

---

## Features

- **One-command access** — `curl | bash` to authenticate, create container, and SSH in. Zero user config
- **Claude Code ready** — Pre-installed in every container. All API requests auto-routed through designated exit IP
- **Full-tunnel egress** — WireGuard + Linux netns / sing-box tun dual-channel, nftables default-deny, no DNS/WebRTC leaks
- **Multi-protocol** — WireGuard and 5 proxy protocols (SOCKS5 / VMess / Shadowsocks / Trojan / HTTP)
- **Per-user isolation** — Dedicated Docker containers with KasmVNC remote desktop + Chromium
- **Admin dashboard** — React SPA for users, hosts, egress IPs, events, and stats
- **User self-service** — View host status, rebuild hosts, access VNC desktop
- **Auto expiration** — Auto-stop containers and block login on expiry
- **Multi-arch CI/CD** — GitHub Actions builds `linux/amd64` + `linux/arm64` images

---

## Deployment

### Docker Compose

```bash
git clone https://github.com/ZaneL1u/cloud-cli-proxy.git
cd cloud-cli-proxy

bash deploy/scripts/setup-env.sh

docker compose up -d --build

curl http://127.0.0.1:8080/healthz
# {"status":"ok"}
```

`setup-env.sh` interactively generates all passwords and secrets. Supports built-in Docker PostgreSQL (zero-config) or external database.

Admin dashboard at `http://YOUR_HOST:3000`, API at `:8080`.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string (required) | — |
| `ADMIN_USERNAME` | Admin username | `admin` |
| `ADMIN_PASSWORD` | Admin password (required) | — |
| `ADMIN_JWT_SECRET` | JWT signing secret (required) | — |
| `ADMIN_PORT` | Admin dashboard port | `3000` |
| `SSH_PROXY_PORT` | SSH proxy port | `2222` |
| `LOG_FORMAT` | Log format `json` / `text` | `json` |
| `LOG_LEVEL` | Log level | `info` |

---

## Usage

### Admin Setup

Log into the admin dashboard, then:

1. **Add egress IPs** — WireGuard config or proxy protocol, with one-click connectivity test
2. **Create users** — Set username, password, expiration
3. **Create hosts** — Create container for user and bind egress IP
4. **Share access command** — Copy the `curl` command from host detail page

### User Access

Users run the command provided by admin:

```bash
curl -sSf http://YOUR_HOST/entry/abc123 | bash
# Enter password → wait for boot → auto SSH into cloud host
```

### Claude Code

Claude Code is pre-installed. Just use it:

```bash
claude
```

All Claude API requests are automatically routed through the designated exit IP. No proxy configuration needed.

### KasmVNC Remote Desktop

Containers include KasmVNC + Chromium. Access the browser desktop via admin or user panel.

---

## Architecture

```
User ──curl──> Control Plane (:8080) ──Docker──> User Container (SSH + Claude Code + VNC)
                    │                                  │
               PostgreSQL                        WireGuard / sing-box Tunnel
                    │                                  │
              Admin SPA (:3000)                  Designated Exit IP
                    │
              SSH Proxy (:2222)
```

| Component | Description |
|-----------|-------------|
| **Control Plane** | Go API — auth, user management, task orchestration, SSH proxy |
| **Host Agent** | Privileged agent — Docker containers, network namespaces, tunnels |
| **User Container** | Ubuntu 24.04 — OpenSSH + Claude Code + KasmVNC + Chromium |
| **PostgreSQL** | Persists users, hosts, egress IPs, tasks, and events |
| **Admin SPA** | React 19 + TypeScript + Vite + Tailwind CSS |

---

## Development

```bash
make setup    # Install deps
make db       # Start PostgreSQL
make dev      # Backend + frontend hot reload
make test     # Run tests
```

See `make help` for all commands.

---

## Documentation

Full docs on [GitHub Pages](https://zanel1u.github.io/cloud-cli-proxy/en/):

- [Quick Start](https://zanel1u.github.io/cloud-cli-proxy/en/guide/quickstart) — Deploy and first use
- [Deployment](https://zanel1u.github.io/cloud-cli-proxy/en/guide/deployment) — systemd native deployment
- [Configuration](https://zanel1u.github.io/cloud-cli-proxy/en/guide/configuration) — Environment variables and WireGuard setup
- [Architecture](https://zanel1u.github.io/cloud-cli-proxy/en/guide/architecture) — System design and project structure
- [API Reference](https://zanel1u.github.io/cloud-cli-proxy/en/reference/api) — Full Admin API
- [FAQ & Recovery](https://zanel1u.github.io/cloud-cli-proxy/en/reference/faq) — Troubleshooting and disaster recovery

---

## License

[MIT](LICENSE)
