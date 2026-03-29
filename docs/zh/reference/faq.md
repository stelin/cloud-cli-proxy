# 故障排查与恢复

## 常见故障

### 控制面无法启动

**症状：** `systemctl status cloud-cli-proxy-control-plane` 显示 `failed`，服务反复重启。

**排查步骤：**

1. 查看日志：`journalctl -u cloud-cli-proxy-control-plane --no-pager -n 50`
2. 检查 `DATABASE_URL`：`grep DATABASE_URL /etc/cloud-cli-proxy/env`
3. 检查 PostgreSQL：`systemctl status postgresql`
4. 检查端口占用：`ss -tlnp | grep 8080`
5. 检查 JWT 密钥：`grep ADMIN_JWT_SECRET /etc/cloud-cli-proxy/env`

**恢复：**
- 数据库连接失败 -> 修复 `DATABASE_URL` 或启动 PostgreSQL
- 端口占用 -> 停止占用进程或修改 `CONTROL_PLANE_ADDR`
- 权限不足 -> 确认 `cloudproxy` 用户有数据库访问权限

### 用户无法登录

**症状：** bootstrap 脚本提示"连接控制面失败"或"认证失败"。

**排查步骤：**

1. 确认控制面运行：`curl -s http://127.0.0.1:8080/healthz`
2. 检查用户状态是否为 `active`
3. 检查用户是否已过期
4. 检查网络连通性（客户端到宿主机 8080 端口）

**恢复：**
- 控制面未运行 -> `systemctl start cloud-cli-proxy-control-plane`
- 用户被禁用 -> 通过 Admin API 重新启用
- 用户已过期 -> 更新到期时间并改回 `active`

### 主机启动失败

**症状：** bootstrap 脚本显示"启动失败"，任务状态为 `failed`。

**排查步骤：**

1. 查看任务详情：通过 Admin API 获取 tasks
2. 检查 Docker daemon：`docker info`
3. 检查受管镜像：`docker images | grep managed-user`
4. 检查磁盘空间：`df -h /var/lib/docker`
5. 检查 host-agent 日志：`journalctl -u cloud-cli-proxy-host-agent --no-pager -n 50`

**恢复：**
- Docker 未运行 -> `systemctl start docker`
- 受管镜像丢失 -> `bash deploy/docker/managed-user/build-managed-image.sh`
- 磁盘空间不足 -> `docker system prune`

### 网络校验失败

**症状：** 出口 IP 不匹配或流量泄漏。

**排查步骤：**

1. 检查出口 IP 绑定状态
2. 检查 WireGuard 接口：`wg show`
3. 检查 nftables 规则：`nft list ruleset`
4. 从容器命名空间手动校验：`nsenter --net=/var/run/netns/cloudproxy-{hostID} curl -s https://api.ipify.org`
5. 检查 DNS 解析：`nsenter --net=/var/run/netns/cloudproxy-{hostID} nslookup example.com`

**恢复：**
- 出口 IP 未绑定 -> 通过 Admin API 创建绑定
- WireGuard 隧道断开 -> 检查 VPN 端点可达性，必要时重建主机
- 防火墙规则异常 -> 重启 host-agent

### 到期扫描未触发

**症状：** 用户已过期但状态仍为 `active`。

**恢复：**
- 控制面未运行 -> 启动控制面
- 需要立即生效 -> `systemctl restart cloud-cli-proxy-control-plane`
- 手动处理 -> 通过 Admin API 手动禁用用户并停止主机

### 数据库连接耗尽

**症状：** 日志出现 `too many connections` 错误。

**排查：**

```bash
sudo -u postgres psql -c "SELECT count(*) FROM pg_stat_activity WHERE datname='cloudproxy'"
sudo -u postgres psql -c "SHOW max_connections"
```

**恢复：**
- 临时释放 -> 重启控制面
- 调整上限 -> 编辑 `postgresql.conf` 增大 `max_connections`

### host-agent 无法启动

**排查步骤：**

1. 查看日志：`journalctl -u cloud-cli-proxy-host-agent --no-pager -n 50`
2. 运行依赖检查：`sudo bash deploy/scripts/host-preflight.sh`
3. 检查 Docker 和 WireGuard
4. 检查 socket 目录权限：`ls -la /run/cloud-cli-proxy/`

## 灾难恢复

### 完全恢复流程

适用于宿主机完全不可用、需要在新机器上恢复的场景。

1. 准备新宿主机，确保满足前置条件
2. 运行自动化部署：
   ```bash
   git clone https://github.com/ZaneL1u/cloud-cli-proxy.git /opt/cloud-cli-proxy
   cd /opt/cloud-cli-proxy
   sudo bash deploy/scripts/deploy.sh
   ```
3. 恢复数据库：
   ```bash
   systemctl stop cloud-cli-proxy-control-plane
   pg_restore --clean -d cloudproxy /path/to/backup.dump
   systemctl start cloud-cli-proxy-control-plane
   ```
4. 重建受管镜像：`bash deploy/docker/managed-user/build-managed-image.sh`
5. 验证服务：`curl -s http://127.0.0.1:8080/healthz`

### 仅数据库恢复

```bash
systemctl stop cloud-cli-proxy-control-plane
sudo -u postgres psql -c "DROP DATABASE cloudproxy"
sudo -u postgres psql -c "CREATE DATABASE cloudproxy OWNER cloudproxy"
pg_restore -d cloudproxy /var/backups/cloud-cli-proxy/最新备份.dump
systemctl start cloud-cli-proxy-control-plane
```

## 日志查看

```bash
# 控制面日志
journalctl -u cloud-cli-proxy-control-plane -f

# host-agent 日志
journalctl -u cloud-cli-proxy-host-agent -f

# 最近 N 行
journalctl -u cloud-cli-proxy-control-plane --no-pager -n 100

# 按时间范围
journalctl -u cloud-cli-proxy-control-plane --since "2026-03-27 00:00:00" --until "2026-03-27 23:59:59"
```
