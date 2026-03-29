# 快速开始

## Docker Compose 部署（推荐）

### 第 1 步：克隆代码

```bash
git clone https://github.com/ZaneL1u/cloud-cli-proxy.git
cd cloud-cli-proxy
```

### 第 2 步：生成环境配置

运行初始化脚本，自动生成所有密码和密钥：

```bash
bash deploy/scripts/setup-env.sh
```

脚本会让你选择数据库方案：

- **内置 Docker PostgreSQL（推荐）**：自动生成数据库密码，Docker Compose 一起管理，零配置。
- **外部 PostgreSQL**：交互式填入外部数据库的地址、端口、用户名、密码，支持 SSL。

两种方案都会自动生成管理员密码（20 位）和 JWT 密钥（48 位）。

### 第 3 步：启动服务

```bash
# 内置 Docker PostgreSQL
docker compose up -d --build

# 外部 PostgreSQL（跳过内置数据库）
docker compose up -d --build control-plane admin
```

### 第 4 步：验证

```bash
curl http://127.0.0.1:8080/healthz
# {"status":"ok"}
```

服务地址：
- API: `http://YOUR_HOST:8080`
- 管理后台: `http://YOUR_HOST:3000`

## 给用户开机器

完整流程分 4 步：**登录 -> 添加出口 IP -> 创建用户 -> 发给用户连接命令**。

### 获取管理员 Token

```bash
TOKEN=$(curl -s -X POST http://YOUR_HOST:8080/v1/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"你的管理员密码"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
```

### 添加出口 IP

```bash
curl -s -X POST http://YOUR_HOST:8080/v1/admin/egress-ips \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "label": "hk-exit-01",
    "ip_address": "203.0.113.10",
    "provider": "manual",
    "wg_endpoint": "vpn-provider.example.com:51820",
    "wg_public_key": "对端公钥Base64",
    "wg_allowed_ips": "0.0.0.0/0",
    "wg_peer_address": "10.0.0.2/32"
  }'
```

### 创建用户

```bash
curl -s -X POST http://YOUR_HOST:8080/v1/admin/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"zhangsan","password":"给用户的初始密码"}'
```

### 发给用户

把下面这条命令发给用户：

```bash
curl -sSf http://YOUR_HOST:8080/v1/bootstrap/script | bash
```

用户输入用户名和密码后，几秒即可进入专属云主机。

## 用户使用方式

> 可以直接把这一段发给你的用户。

在终端执行以下命令，输入管理员给你的用户名和密码：

```bash
curl -sSf http://YOUR_HOST:8080/v1/bootstrap/script | bash
```

容器内已预装：
- **Claude Code** -- 直接在终端使用
- **浏览器桌面** -- 通过 VNC 访问（端口 6080）
- 常用工具：Git、tmux、zsh、Node.js 等

如果连接中断，重新执行同一条 `curl` 命令即可恢复。

## 下一步

- [部署指南](./deployment) -- systemd 原生部署
- [配置参考](./configuration) -- 环境变量和 WireGuard 配置
- [API 参考](../reference/api) -- 完整的 Admin API 文档
