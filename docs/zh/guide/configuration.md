# 配置参考

## 环境变量

创建 `/etc/cloud-cli-proxy/env`（systemd 部署）或 `.env`（Docker Compose 部署）：

| 变量 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `DATABASE_URL` | 是 | -- | PostgreSQL 连接字符串 |
| `CONTROL_PLANE_ADDR` | 否 | `:8080` | 控制面监听地址 |
| `ADMIN_USERNAME` | 否 | `admin` | 管理员用户名 |
| `ADMIN_PASSWORD` | 建议 | -- | 管理员密码（未设置时管理后台可能不安全） |
| `ADMIN_JWT_SECRET` | 是 | -- | JWT 签名密钥（至少 32 字符），未设置则禁用 admin API |
| `HOST_AGENT_SOCKET` | 否 | `/run/cloud-cli-proxy/host-agent.sock` | host-agent Unix socket 路径 |

使用 `setup-env.sh` 可以交互式生成配置：

```bash
bash deploy/scripts/setup-env.sh
```

## WireGuard 配置

每个出口 IP 对应一个 WireGuard peer。通过 Admin API 创建出口 IP 时需提供以下参数：

| 参数 | 说明 |
|------|------|
| `wg_endpoint` | WireGuard peer 端点（如 `1.2.3.4:51820`） |
| `wg_public_key` | peer 公钥 |
| `wg_preshared_key` | 预共享密钥（可选） |
| `wg_allowed_ips` | 允许的 IP 范围（默认 `0.0.0.0/0`，全隧道） |
| `wg_dns_server` | DNS 服务器（可选） |
| `wg_peer_address` | 本端分配的地址（CIDR 格式） |

WireGuard 接口在容器创建时由 host-agent 自动配置到容器的网络命名空间。

## 防火墙规则

host-agent 使用 nftables 为每个容器的网络命名空间设置默认拒绝策略，仅允许通过 WireGuard 隧道出网。规则由 host-agent 自动管理，无需手动配置。

宿主机防火墙建议：

```bash
nft add table inet filter
nft add chain inet filter input '{ type filter hook input priority 0; policy drop; }'
nft add rule inet filter input ct state established,related accept
nft add rule inet filter input iif lo accept
nft add rule inet filter input tcp dport 22 accept
nft add rule inet filter input tcp dport 8080 accept
```

## Docker 镜像

所有镜像通过 GitHub Actions 自动构建，支持 `linux/amd64` 和 `linux/arm64`。

| 镜像 | 地址 |
|------|------|
| control-plane | `ghcr.io/zanel1u/cloud-cli-proxy/control-plane` |
| admin | `ghcr.io/zanel1u/cloud-cli-proxy/admin` |
| managed-user | `ghcr.io/zanel1u/cloud-cli-proxy/managed-user` |
| sing-box-gateway | `ghcr.io/zanel1u/cloud-cli-proxy/sing-box-gateway` |

**镜像标签规则：**

| 标签 | 说明 |
|------|------|
| `latest` | main 分支最新构建 |
| `1.2.3` | 发布版本，对应 GitHub Release |
| `1.2` | 自动跟随最新 patch |
| `1` | 自动跟随最新 minor |
| `a1b2c3d` | 精确到提交 |

**生产环境建议锁定版本：**

```bash
docker pull ghcr.io/zanel1u/cloud-cli-proxy/control-plane:1.2.3
```
