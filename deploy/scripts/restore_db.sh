#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "usage: restore_db.sh <backup.sql|backup.sql.gz>"
    exit 1
fi

ENV_FILE="${ENV_FILE:-/etc/sitecraft/sitecraft.env}"
BACKUP_FILE="$1"

if [[ ! -f "$ENV_FILE" ]]; then
    echo "missing env file: $ENV_FILE"
    exit 1
fi
if [[ ! -f "$BACKUP_FILE" ]]; then
    echo "missing backup file: $BACKUP_FILE"
    exit 1
fi

set -a
source "$ENV_FILE"
set +a

if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "DATABASE_URL is required"
    exit 1
fi

if [[ "$BACKUP_FILE" == *.gz ]]; then
    gzip -dc "$BACKUP_FILE" | psql "$DATABASE_URL" -v ON_ERROR_STOP=1
else
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 < "$BACKUP_FILE"
fi
