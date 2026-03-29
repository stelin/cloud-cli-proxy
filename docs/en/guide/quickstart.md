# Quick Start

## Docker Compose Deployment (Recommended)

### Step 1: Clone

```bash
git clone https://github.com/ZaneL1u/cloud-cli-proxy.git
cd cloud-cli-proxy
```

### Step 2: Generate Environment Config

Run the setup script to auto-generate all passwords and secrets:

```bash
bash deploy/scripts/setup-env.sh
```

Choose a database mode:

- **Built-in Docker PostgreSQL (recommended)**: auto-generates DB password, managed by Docker Compose, zero config.
- **External PostgreSQL**: interactively enter your DB host, port, credentials, with SSL support.

Both options auto-generate an admin password (20 chars) and JWT secret (48 chars).

### Step 3: Start Services

```bash
# Built-in Docker PostgreSQL
docker compose up -d --build

# External PostgreSQL (skip built-in DB)
docker compose up -d --build control-plane admin
```

### Step 4: Verify

```bash
curl http://127.0.0.1:8080/healthz
# {"status":"ok"}
```

Service endpoints:
- API: `http://YOUR_HOST:8080`
- Admin dashboard: `http://YOUR_HOST:3000`

## Provisioning Users

Four steps: **login -> add egress IP -> create user -> send connection command**.

### Get Admin Token

```bash
TOKEN=$(curl -s -X POST http://YOUR_HOST:8080/v1/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-admin-password"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
```

### Add Egress IP

```bash
curl -s -X POST http://YOUR_HOST:8080/v1/admin/egress-ips \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "label": "hk-exit-01",
    "ip_address": "203.0.113.10",
    "provider": "manual",
    "wg_endpoint": "vpn-provider.example.com:51820",
    "wg_public_key": "PeerPublicKeyBase64",
    "wg_allowed_ips": "0.0.0.0/0",
    "wg_peer_address": "10.0.0.2/32"
  }'
```

### Create User

```bash
curl -s -X POST http://YOUR_HOST:8080/v1/admin/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"zhangsan","password":"initial-password-for-user"}'
```

### Send to User

Share this command with your user:

```bash
curl -sSf http://YOUR_HOST:8080/v1/bootstrap/script | bash
```

After entering username and password, they'll land in their dedicated cloud machine within seconds.

## User Access

> Share this section directly with your users.

Run the following command and enter the credentials provided by your admin:

```bash
curl -sSf http://YOUR_HOST:8080/v1/bootstrap/script | bash
```

Pre-installed tools:
- **Claude Code** -- use directly in terminal
- **Browser desktop** -- accessible via VNC (port 6080)
- Common tools: Git, tmux, zsh, Node.js, etc.

If disconnected, re-run the same `curl` command to reconnect.

## Next Steps

- [Deployment Guide](./deployment) -- systemd native deployment
- [Configuration](./configuration) -- Environment variables and WireGuard setup
- [API Reference](../reference/api) -- Full Admin API docs
