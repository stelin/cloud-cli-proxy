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

**Go · React · PostgreSQL · Docker · sing-box**

</div>

---

## Features

- **One-command access** — `curl | bash` to authenticate, create container, and SSH in. Zero user config
- **cloud-claude local CLI** — `alias claude=cloud-claude` to run remote Claude Code from your terminal; local cwd is sshfs-mounted at the **same path** in the container; optional local exec for commands like `git`
- **Claude Code ready** — Pre-installed in every container. All API requests auto-routed through designated exit IP
- **Full-tunnel egress** — sing-box tun + Linux netns full-tunnel, nftables default-deny, no DNS/WebRTC leaks
- **Multi-protocol** — 6 proxy protocols (SOCKS5 / VMess / VLESS / Shadowsocks / Trojan / HTTP)
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

1. **Add egress IPs** — Multiple proxy protocols, with one-click connectivity test
2. **Create users** — Set username, password, expiration
3. **Create hosts** — Create container for user and bind egress IP
4. **Share access info** — Copy the `curl` command from host details; for `cloud-claude` users also share: **gateway HTTPS URL**, **host Short ID**, and **user password**

### User Access

Users run the command provided by admin:

```bash
curl -sSf http://YOUR_HOST/entry/abc123 | bash
# Enter password → wait for boot → auto SSH into cloud host
```

### cloud-claude (local CLI, recommended)

After the admin **creates the host, binds an egress IP**, and the container is ready, give the user three things:

| Field | Meaning |
|-------|---------|
| **Gateway URL** | Public HTTPS base URL of the control plane, e.g. `https://gw.example.com` (same origin you use for the admin UI in the browser; usually **not** the `:3000` admin dev port) |
| **Short ID** | **Host** short ID from the host detail page. If the user configures a **user** short ID instead, they connect to that user’s primary host |
| **Password** | The user’s password from the admin dashboard |

Install the CLI once, run `init`, then from **any project directory** run `cloud-claude` — the cwd is mounted at the **same path** in the container. By default `git` runs locally (tune with `proxy_commands` in `~/.cloud-claude/config.yaml`).

#### Install cloud-claude

**Homebrew (macOS / Linux, recommended):**

```bash
brew tap ZaneL1u/tap
brew install cloud-claude
```

**One-liner (any platform):**

```bash
curl -fsSL https://raw.githubusercontent.com/ZaneL1u/cloud-cli-proxy/main/scripts/install.sh | bash
```

Or download the matching `tar.gz` from [Releases](https://github.com/ZaneL1u/cloud-cli-proxy/releases), or build from source:

```bash
go build -ldflags "-s -w" -trimpath -o cloud-claude ./cmd/cloud-claude
```

#### First-time setup

```bash
cloud-claude init
# Prompts: gateway, Short ID, password → ~/.cloud-claude/config.yaml
```

Flags or environment variables:

```bash
cloud-claude init --gateway https://gw.example.com --short-id abc123 --password your-password

export CLOUD_CLAUDE_GATEWAY=https://gw.example.com
export CLOUD_CLAUDE_SHORT_ID=abc123
export CLOUD_CLAUDE_PASSWORD=your-password
cloud-claude init
```

#### Daily use

```bash
cd ~/your/project   # repo root you want Claude Code to see

alias claude=cloud-claude   # optional

cloud-claude
cloud-claude -p "refactor this function"
```

**Optional:** verify remote timezone, locale, egress IP, FUSE, etc.:

```bash
cloud-claude env check
```

**Optional:** set `proxy_commands` in `~/.cloud-claude/config.yaml` (list of command names to run on the host). Default is `git` only; use an empty list to disable.

`cloud-claude` does: gateway auth → wait for container → sshfs mount at the same path → start Claude Code remotely. Terminal size, signals, and exit codes are forwarded.

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
               PostgreSQL                            │  sshfs ← same path as local cwd  │
                    │                                │  sing-box tun Tunnel              │
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
| **cloud-claude** | Go CLI — transparent `claude`; sshfs same-path mount; optional local command proxy |
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
- [Configuration](https://zanel1u.github.io/cloud-cli-proxy/en/guide/configuration) — Environment variables and egress proxy setup
- [Architecture](https://zanel1u.github.io/cloud-cli-proxy/en/guide/architecture) — System design and project structure
- [API Reference](https://zanel1u.github.io/cloud-cli-proxy/en/reference/api) — Full Admin API
- [FAQ & Recovery](https://zanel1u.github.io/cloud-cli-proxy/en/reference/faq) — Troubleshooting and disaster recovery

---

## License

[MIT](LICENSE)
