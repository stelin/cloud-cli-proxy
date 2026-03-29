#!/usr/bin/env bash
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/cloud-cli-proxy}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
DB_NAME="${DB_NAME:-cloudproxy}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/${DB_NAME}_${TIMESTAMP}.dump"

mkdir -p "$BACKUP_DIR"

echo "[backup] Starting PostgreSQL backup: ${DB_NAME}"
pg_dump -Fc "$DB_NAME" > "$BACKUP_FILE"
echo "[backup] Backup saved: $BACKUP_FILE ($(du -h "$BACKUP_FILE" | cut -f1))"

find "$BACKUP_DIR" -name "${DB_NAME}_*.dump" -mtime +"$RETENTION_DAYS" -delete
echo "[backup] Cleaned backups older than ${RETENTION_DAYS} days"
