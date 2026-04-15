<div align="center">

<img src="docs/public/logo.svg" width="88" height="88" alt="Cloud CLI Proxy" />

# Cloud CLI Proxy

**One command. One cloud machine. All traffic through your exit IP.**

Out-of-the-box isolated cloud hosts for Claude Code and dev teams. Pre-installed AI coding tools, full-tunnel egress through designated IPs, zero leaks.

[![CI](https://github.com/ZaneL1u/cloud-cli-proxy/actions/workflows/ci.yml/badge.svg)](https://github.com/ZaneL1u/cloud-cli-proxy/actions/workflows/ci.yml)
[![Images](https://github.com/ZaneL1u/cloud-cli-proxy/actions/workflows/build-images.yml/badge.svg)](https://github.com/ZaneL1u/cloud-cli-proxy/actions/workflows/build-images.yml)
[![Release](https://img.shields.io/github/v/release/ZaneL1u/cloud-cli-proxy)](https://github.com/ZaneL1u/cloud-cli-proxy/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[中文](README.md) | [Documentation](https://zanel1u.github.io/cloud-cli-proxy/en/)

**Go · React · PostgreSQL · Docker · WireGuard**

</div>

---

## Features

- **One-command access** — `curl | bash` to authenticate, create container, and SSH in. Zero user config
- **cloud-claude local CLI** — `alias claude=cloud-claude` to transparently run remote Claude Code from your local terminal with real-time directory mapping
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

# Recommended: prefer prebuilt images (latest)
docker compose pull --policy always
docker compose up -d

curl http://127.0.0.1:8080/healthz
# {"status":"ok"}
```

`setup-env.sh` interactively generates all passwords and secrets. Supports built-in Docker PostgreSQL (zero-config) or external database.

Admin dashboard at `http://YOUR_HOST:3000`, API at `:8080`.

Optional local source build (fallback when prebuilt images are unavailable):

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yaml --profile build-only build --no-cache
docker compose -f docker-compose.yml -f docker-compose.build.yaml up -d --force-recreate
```

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

### cloud-claude (Local CLI, Transparent Remote)

Besides SSH access, you can use the `cloud-claude` binary on your local machine to transparently run remote Claude Code with your current directory auto-mapped into the container.

**Install:**

Download from [Releases](https://github.com/ZaneL1u/cloud-cli-proxy/releases), or build from source:

```bash
go build -o cloud-claude ./cmd/cloud-claude
```

**Initialize config:**

```bash
cloud-claude init
# Interactive prompts:
#   Gateway URL (e.g. https://gw.example.com)
#   Short ID (host short ID from admin)
#   Password
# Saved to ~/.cloud-claude/config.yaml
```

You can also pass values via flags or environment variables:

```bash
# Flags
cloud-claude init --gateway https://gw.example.com --short-id abc123 --password your-password

# Environment variables
export CLOUD_CLAUDE_GATEWAY=https://gw.example.com
export CLOUD_CLAUDE_SHORT_ID=abc123
export CLOUD_CLAUDE_PASSWORD=your-password
cloud-claude init
```

**Usage:**

```bash
# Set alias so `claude` transparently uses the remote host
alias claude=cloud-claude

# Use exactly like local claude
claude

# All claude arguments are passed through
claude -p "refactor this function"
claude --model sonnet
```

`cloud-claude` automatically: authenticates → waits for container ready → maps your current directory to `/workspace` via sshfs → launches Claude Code remotely. Terminal resizing, signals, and exit codes are all properly forwarded.

### Claude Code (via SSH)

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
                                                    ┌───────────────────────────────────┐
User ──curl──> Control Plane (:8080) ──Docker──>    │ User Container                    │
                    │                                │  SSH + Claude Code + VNC          │
               PostgreSQL                            │  sshfs ← /workspace dir mapping  │
                    │                                │  WireGuard / sing-box Tunnel      │
              Admin SPA (:3000)                      │       ↓                           │
                    │                                │  Designated Exit IP               │
              SSH Proxy (:2222)                      └───────────────────────────────────┘
                    ↑                                           ↑
                    │                                           │
User ──cloud-claude──> auth + SSH + sshfs ──────────────────────┘
```

| Component | Description |
|-----------|-------------|
| **Control Plane** | Go API — auth, user management, task orchestration, SSH proxy |
| **Host Agent** | Privileged agent — Docker containers, network namespaces, tunnels |
| **User Container** | Ubuntu 24.04 — OpenSSH + Claude Code + sshfs + KasmVNC + Chromium |
| **cloud-claude** | Go CLI — transparent local replacement for `claude`, maps local dir via sshfs |
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

## Release And Changelog

Pushing a `v*` tag triggers the `Release` workflow automatically and does three things:

- Runs CI quality gates first (Go tests + admin web build)
- Creates a GitHub Release
- Publishes multi-arch images (`semver` + `latest`)
- Generates monorepo-grouped release notes and writes them to [CHANGELOG.md](CHANGELOG.md)

The default changelog groups are path-based:

- Backend (Go / API, `cmd` + `internal`)
- Frontend (`web/admin`)
- Runtime & Deployment (`deploy`, compose files, workflows)
- Docs (`docs` + READMEs)

Manual release example:

```bash
make release VERSION=1.5.0
```

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
