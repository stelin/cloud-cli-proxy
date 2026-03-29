# API 参考

## 认证

所有 Admin API 调用需要先获取 JWT Token：

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/v1/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"你的管理员密码"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
```

后续命令均使用 `Authorization: Bearer $TOKEN` 作为认证凭证。

## 用户管理

### 创建用户

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"newuser","password":"初始密码至少8位"}'
```

用户名长度 3-50 字符，密码至少 8 字符。创建成功返回 `201`，用户名冲突返回 `409`。

### 查看用户列表

```bash
curl -s http://127.0.0.1:8080/v1/admin/users \
  -H "Authorization: Bearer $TOKEN"
```

### 查看单个用户详情

```bash
curl -s http://127.0.0.1:8080/v1/admin/users/{userID} \
  -H "Authorization: Bearer $TOKEN"
```

### 禁用 / 启用用户

```bash
curl -s -X PATCH http://127.0.0.1:8080/v1/admin/users/{userID} \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"disabled"}'
```

可选值：`active`、`disabled`。

### 删除用户

```bash
curl -s -X DELETE http://127.0.0.1:8080/v1/admin/users/{userID} \
  -H "Authorization: Bearer $TOKEN"
```

成功返回 `204`。关联的主机会因外键级联删除。

### 密码轮换

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/users/{userID}/rotate-password \
  -H "Authorization: Bearer $TOKEN"
```

系统自动生成 20 字符随机强密码，返回新密码明文。

### 设置到期时间

```bash
curl -s -X PUT http://127.0.0.1:8080/v1/admin/users/{userID}/expiry \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expires_at":"2026-12-31T23:59:59Z"}'
```

清除到期时间（永不过期）：

```bash
curl -s -X PUT http://127.0.0.1:8080/v1/admin/users/{userID}/expiry \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expires_at":null}'
```

## 出口 IP 管理

### 创建出口 IP

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/egress-ips \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "label": "exit-node-1",
    "ip_address": "203.0.113.10",
    "provider": "manual",
    "wg_endpoint": "vpn-provider.example.com:51820",
    "wg_public_key": "peer公钥Base64",
    "wg_preshared_key": "预共享密钥Base64（可选）",
    "wg_allowed_ips": "0.0.0.0/0",
    "wg_dns_server": "1.1.1.1",
    "wg_peer_address": "10.0.0.2/32"
  }'
```

### 查看出口 IP 列表

```bash
curl -s http://127.0.0.1:8080/v1/admin/egress-ips \
  -H "Authorization: Bearer $TOKEN"
```

### 更新出口 IP

```bash
curl -s -X PUT http://127.0.0.1:8080/v1/admin/egress-ips/{ipID} \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"wg_endpoint": "新端点:51820", "wg_public_key": "新公钥"}'
```

### 删除出口 IP

```bash
curl -s -X DELETE http://127.0.0.1:8080/v1/admin/egress-ips/{ipID} \
  -H "Authorization: Bearer $TOKEN"
```

如果该 IP 仍被主机绑定，删除会被拒绝。需先解除绑定再删除。

## 主机与出口 IP 绑定

### 创建绑定

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/bindings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"host_id":"主机UUID","egress_ip_id":"出口IP的UUID"}'
```

### 解除绑定

```bash
curl -s -X DELETE http://127.0.0.1:8080/v1/admin/bindings/{bindingID} \
  -H "Authorization: Bearer $TOKEN"
```

## 主机运维

### 查看主机列表

```bash
curl -s http://127.0.0.1:8080/v1/admin/hosts \
  -H "Authorization: Bearer $TOKEN"
```

### 启动 / 停止主机

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/hosts/{hostID}/start \
  -H "Authorization: Bearer $TOKEN"

curl -s -X POST http://127.0.0.1:8080/v1/admin/hosts/{hostID}/stop \
  -H "Authorization: Bearer $TOKEN"
```

### 重建主机

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/hosts/{hostID}/rebuild \
  -H "Authorization: Bearer $TOKEN"
```

重建会销毁当前容器并基于受管镜像重新创建，用户 home volume 会保留。

### 查看任务状态

```bash
curl -s http://127.0.0.1:8080/v1/admin/tasks \
  -H "Authorization: Bearer $TOKEN"
```

## 事件查看

```bash
# 最近事件
curl -s "http://127.0.0.1:8080/v1/admin/events?limit=50" \
  -H "Authorization: Bearer $TOKEN"

# 按用户筛选
curl -s "http://127.0.0.1:8080/v1/admin/events?user_id={userID}" \
  -H "Authorization: Bearer $TOKEN"

# 按主机筛选
curl -s "http://127.0.0.1:8080/v1/admin/events?host_id={hostID}" \
  -H "Authorization: Bearer $TOKEN"

# 按时间范围
curl -s "http://127.0.0.1:8080/v1/admin/events?since=2026-03-01T00:00:00Z&until=2026-03-31T23:59:59Z" \
  -H "Authorization: Bearer $TOKEN"
```

## 仪表盘

```bash
curl -s http://127.0.0.1:8080/v1/admin/dashboard/stats \
  -H "Authorization: Bearer $TOKEN"
```

## 备份与恢复

### 数据库备份

```bash
sudo bash deploy/scripts/backup.sh
```

默认配置：备份目录 `/var/backups/cloud-cli-proxy`，保留 7 天。

通过环境变量自定义：

```bash
BACKUP_DIR=/data/backups RETENTION_DAYS=30 bash deploy/scripts/backup.sh
```

建议通过 cron 设置定期备份：

```bash
echo "0 2 * * * root /opt/cloud-cli-proxy/deploy/scripts/backup.sh" > /etc/cron.d/cloud-cli-proxy-backup
```

## 密钥轮换

### JWT Secret 轮换

1. 生成新密钥：`NEW_SECRET=$(head -c 48 /dev/urandom | base64 | tr -d '=+/' | head -c 48)`
2. 更新 `/etc/cloud-cli-proxy/env` 中的 `ADMIN_JWT_SECRET`
3. 重启控制面：`systemctl restart cloud-cli-proxy-control-plane`

轮换后所有已颁发的 JWT Token 立即失效。

### WireGuard 密钥轮换

1. 从出口 IP 提供商获取新密钥对
2. 通过 Admin API 更新出口 IP 的 WireGuard 配置
3. 重启使用该出口 IP 的主机以加载新配置
