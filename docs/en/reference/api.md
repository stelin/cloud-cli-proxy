# API Reference

## Authentication

All Admin API calls require a JWT Token:

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/v1/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-admin-password"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
```

Use `Authorization: Bearer $TOKEN` header for all subsequent requests.

## User Management

### Create User

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"newuser","password":"min-8-chars"}'
```

Username: 3-50 chars. Password: 8+ chars. Returns `201` on success, `409` on conflict.

### List Users

```bash
curl -s http://127.0.0.1:8080/v1/admin/users \
  -H "Authorization: Bearer $TOKEN"
```

### Get User Details

```bash
curl -s http://127.0.0.1:8080/v1/admin/users/{userID} \
  -H "Authorization: Bearer $TOKEN"
```

### Disable / Enable User

```bash
curl -s -X PATCH http://127.0.0.1:8080/v1/admin/users/{userID} \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"disabled"}'
```

Values: `active`, `disabled`.

### Delete User

```bash
curl -s -X DELETE http://127.0.0.1:8080/v1/admin/users/{userID} \
  -H "Authorization: Bearer $TOKEN"
```

Returns `204`. Associated hosts are cascade-deleted.

### Rotate Password

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/users/{userID}/rotate-password \
  -H "Authorization: Bearer $TOKEN"
```

Returns a new auto-generated 20-char password in plaintext.

### Set Expiration

```bash
curl -s -X PUT http://127.0.0.1:8080/v1/admin/users/{userID}/expiry \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expires_at":"2026-12-31T23:59:59Z"}'
```

Clear expiration (never expires):

```bash
curl -s -X PUT http://127.0.0.1:8080/v1/admin/users/{userID}/expiry \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expires_at":null}'
```

## Egress IP Management

### Create Egress IP

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/egress-ips \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "label": "exit-node-1",
    "ip_address": "203.0.113.10",
    "provider": "manual",
    "wg_endpoint": "vpn-provider.example.com:51820",
    "wg_public_key": "PeerPublicKeyBase64",
    "wg_preshared_key": "PresharedKeyBase64 (optional)",
    "wg_allowed_ips": "0.0.0.0/0",
    "wg_dns_server": "1.1.1.1",
    "wg_peer_address": "10.0.0.2/32"
  }'
```

### List Egress IPs

```bash
curl -s http://127.0.0.1:8080/v1/admin/egress-ips \
  -H "Authorization: Bearer $TOKEN"
```

### Update Egress IP

```bash
curl -s -X PUT http://127.0.0.1:8080/v1/admin/egress-ips/{ipID} \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"wg_endpoint": "new-endpoint:51820", "wg_public_key": "NewKey"}'
```

### Delete Egress IP

```bash
curl -s -X DELETE http://127.0.0.1:8080/v1/admin/egress-ips/{ipID} \
  -H "Authorization: Bearer $TOKEN"
```

Rejects deletion if IP is still bound to hosts. Unbind first.

## Host & Egress IP Bindings

### Create Binding

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/bindings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"host_id":"host-uuid","egress_ip_id":"egress-ip-uuid"}'
```

### Remove Binding

```bash
curl -s -X DELETE http://127.0.0.1:8080/v1/admin/bindings/{bindingID} \
  -H "Authorization: Bearer $TOKEN"
```

## Host Operations

### List Hosts

```bash
curl -s http://127.0.0.1:8080/v1/admin/hosts \
  -H "Authorization: Bearer $TOKEN"
```

### Start / Stop Host

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/hosts/{hostID}/start \
  -H "Authorization: Bearer $TOKEN"

curl -s -X POST http://127.0.0.1:8080/v1/admin/hosts/{hostID}/stop \
  -H "Authorization: Bearer $TOKEN"
```

### Rebuild Host

```bash
curl -s -X POST http://127.0.0.1:8080/v1/admin/hosts/{hostID}/rebuild \
  -H "Authorization: Bearer $TOKEN"
```

Destroys and recreates the container from the managed image. Home volume is preserved.

### List Tasks

```bash
curl -s http://127.0.0.1:8080/v1/admin/tasks \
  -H "Authorization: Bearer $TOKEN"
```

## Events

```bash
# Recent events
curl -s "http://127.0.0.1:8080/v1/admin/events?limit=50" \
  -H "Authorization: Bearer $TOKEN"

# Filter by user
curl -s "http://127.0.0.1:8080/v1/admin/events?user_id={userID}" \
  -H "Authorization: Bearer $TOKEN"

# Filter by host
curl -s "http://127.0.0.1:8080/v1/admin/events?host_id={hostID}" \
  -H "Authorization: Bearer $TOKEN"

# Time range
curl -s "http://127.0.0.1:8080/v1/admin/events?since=2026-03-01T00:00:00Z&until=2026-03-31T23:59:59Z" \
  -H "Authorization: Bearer $TOKEN"
```

## Dashboard

```bash
curl -s http://127.0.0.1:8080/v1/admin/dashboard/stats \
  -H "Authorization: Bearer $TOKEN"
```

## Backup & Recovery

### Database Backup

```bash
sudo bash deploy/scripts/backup.sh
```

Defaults: directory `/var/backups/cloud-cli-proxy`, retention 7 days.

Customize:

```bash
BACKUP_DIR=/data/backups RETENTION_DAYS=30 bash deploy/scripts/backup.sh
```

### Key Rotation

**JWT Secret:**
1. Generate: `NEW_SECRET=$(head -c 48 /dev/urandom | base64 | tr -d '=+/' | head -c 48)`
2. Update `ADMIN_JWT_SECRET` in env file
3. Restart: `systemctl restart cloud-cli-proxy-control-plane`

**WireGuard keys:**
1. Get new keys from provider
2. Update via Admin API
3. Restart affected hosts
