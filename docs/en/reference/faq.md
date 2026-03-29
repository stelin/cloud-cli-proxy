# FAQ & Troubleshooting

## Common Issues

### Control Plane Won't Start

**Symptoms:** `systemctl status cloud-cli-proxy-control-plane` shows `failed`, service keeps restarting.

**Check:**

1. View logs: `journalctl -u cloud-cli-proxy-control-plane --no-pager -n 50`
2. Verify `DATABASE_URL`: `grep DATABASE_URL /etc/cloud-cli-proxy/env`
3. Check PostgreSQL: `systemctl status postgresql`
4. Check port conflict: `ss -tlnp | grep 8080`
5. Check JWT secret: `grep ADMIN_JWT_SECRET /etc/cloud-cli-proxy/env`

**Fix:**
- DB connection failure -> Fix `DATABASE_URL` or start PostgreSQL
- Port conflict -> Stop conflicting process or change `CONTROL_PLANE_ADDR`
- Permissions -> Verify `cloudproxy` user has DB access

### User Can't Log In

**Symptoms:** bootstrap script shows "connection failed" or "auth failed".

**Check:**

1. Control plane running: `curl -s http://127.0.0.1:8080/healthz`
2. User status is `active`
3. User not expired
4. Network connectivity (client to host port 8080)

**Fix:**
- Control plane down -> `systemctl start cloud-cli-proxy-control-plane`
- User disabled -> Re-enable via Admin API
- User expired -> Update expiry and set status to `active`

### Host Startup Failure

**Symptoms:** bootstrap shows "startup failed", task status is `failed`.

**Check:**

1. Task details via Admin API
2. Docker daemon: `docker info`
3. Managed image exists: `docker images | grep managed-user`
4. Disk space: `df -h /var/lib/docker`
5. Host-agent logs: `journalctl -u cloud-cli-proxy-host-agent --no-pager -n 50`

**Fix:**
- Docker not running -> `systemctl start docker`
- Image missing -> `bash deploy/docker/managed-user/build-managed-image.sh`
- Disk full -> `docker system prune`

### Network Verification Failure

**Symptoms:** Egress IP mismatch or traffic leaks.

**Check:**

1. Egress IP binding status
2. WireGuard interface: `wg show`
3. nftables rules: `nft list ruleset`
4. Manual verify from namespace: `nsenter --net=/var/run/netns/cloudproxy-{hostID} curl -s https://api.ipify.org`
5. DNS resolution: `nsenter --net=/var/run/netns/cloudproxy-{hostID} nslookup example.com`

**Fix:**
- IP not bound -> Create binding via Admin API
- Tunnel down -> Check VPN endpoint reachability, rebuild host if needed
- Firewall rules broken -> Restart host-agent

### Expiry Scanner Not Running

**Fix:**
- Control plane down -> Start it
- Immediate effect needed -> `systemctl restart cloud-cli-proxy-control-plane`
- Manual override -> Disable user and stop host via Admin API

### Database Connection Exhaustion

**Symptoms:** Logs show `too many connections`.

**Check:**

```bash
sudo -u postgres psql -c "SELECT count(*) FROM pg_stat_activity WHERE datname='cloudproxy'"
sudo -u postgres psql -c "SHOW max_connections"
```

**Fix:**
- Temporary -> Restart control plane
- Permanent -> Increase `max_connections` in `postgresql.conf`

## Disaster Recovery

### Full Recovery

For when the host is completely unavailable and you need to restore on a new machine:

1. Prepare new host meeting prerequisites
2. Deploy:
   ```bash
   git clone https://github.com/ZaneL1u/cloud-cli-proxy.git /opt/cloud-cli-proxy
   cd /opt/cloud-cli-proxy
   sudo bash deploy/scripts/deploy.sh
   ```
3. Restore database:
   ```bash
   systemctl stop cloud-cli-proxy-control-plane
   pg_restore --clean -d cloudproxy /path/to/backup.dump
   systemctl start cloud-cli-proxy-control-plane
   ```
4. Rebuild image: `bash deploy/docker/managed-user/build-managed-image.sh`
5. Verify: `curl -s http://127.0.0.1:8080/healthz`

### Database-only Recovery

```bash
systemctl stop cloud-cli-proxy-control-plane
sudo -u postgres psql -c "DROP DATABASE cloudproxy"
sudo -u postgres psql -c "CREATE DATABASE cloudproxy OWNER cloudproxy"
pg_restore -d cloudproxy /var/backups/cloud-cli-proxy/latest-backup.dump
systemctl start cloud-cli-proxy-control-plane
```

## Logs

```bash
# Control plane (follow)
journalctl -u cloud-cli-proxy-control-plane -f

# Host-agent (follow)
journalctl -u cloud-cli-proxy-host-agent -f

# Last N lines
journalctl -u cloud-cli-proxy-control-plane --no-pager -n 100

# Time range
journalctl -u cloud-cli-proxy-control-plane --since "2026-03-27 00:00:00" --until "2026-03-27 23:59:59"
```
