# Configuration

## Environment Variables

Create `/etc/cloud-cli-proxy/env` (systemd) or `.env` (Docker Compose):

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | -- | PostgreSQL connection string |
| `CONTROL_PLANE_ADDR` | No | `:8080` | Control plane listen address |
| `ADMIN_USERNAME` | No | `admin` | Admin username |
| `ADMIN_PASSWORD` | Recommended | -- | Admin password |
| `ADMIN_JWT_SECRET` | Yes | -- | JWT signing key (32+ chars), disables admin API if unset |
| `HOST_AGENT_SOCKET` | No | `/run/cloud-cli-proxy/host-agent.sock` | Host-agent Unix socket path |

Use `setup-env.sh` for interactive generation:

```bash
bash deploy/scripts/setup-env.sh
```

## WireGuard Configuration

Each egress IP corresponds to a WireGuard peer. Provide these parameters when creating an egress IP via Admin API:

| Parameter | Description |
|-----------|-------------|
| `wg_endpoint` | WireGuard peer endpoint (e.g., `1.2.3.4:51820`) |
| `wg_public_key` | Peer public key |
| `wg_preshared_key` | Pre-shared key (optional) |
| `wg_allowed_ips` | Allowed IP range (default `0.0.0.0/0`, full tunnel) |
| `wg_dns_server` | DNS server (optional) |
| `wg_peer_address` | Local assigned address (CIDR format) |

WireGuard interfaces are automatically configured by host-agent into the container's network namespace on creation.

## Firewall Rules

Host-agent uses nftables to set default-deny policy for each container's namespace, allowing only WireGuard tunnel egress. Rules are managed automatically.

Recommended host firewall:

```bash
nft add table inet filter
nft add chain inet filter input '{ type filter hook input priority 0; policy drop; }'
nft add rule inet filter input ct state established,related accept
nft add rule inet filter input iif lo accept
nft add rule inet filter input tcp dport 22 accept
nft add rule inet filter input tcp dport 8080 accept
```

## Docker Images

All images are built via GitHub Actions for `linux/amd64` and `linux/arm64`.

| Image | Registry |
|-------|----------|
| control-plane | `ghcr.io/zanel1u/cloud-cli-proxy/control-plane` |
| admin | `ghcr.io/zanel1u/cloud-cli-proxy/admin` |
| managed-user | `ghcr.io/zanel1u/cloud-cli-proxy/managed-user` |
| sing-box-gateway | `ghcr.io/zanel1u/cloud-cli-proxy/sing-box-gateway` |

**Tag convention:**

| Tag | Description |
|-----|-------------|
| `latest` | Latest build from main |
| `1.2.3` | Release version, corresponds to GitHub Release |
| `1.2` | Auto-follows latest patch |
| `1` | Auto-follows latest minor |
| `a1b2c3d` | Pinned to exact commit |

**Pin versions in production:**

```bash
docker pull ghcr.io/zanel1u/cloud-cli-proxy/control-plane:1.2.3
```
