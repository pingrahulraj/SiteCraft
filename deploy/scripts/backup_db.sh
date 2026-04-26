#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${ENV_FILE:-/etc/sitecraft/sitecraft.env}"
OUTPUT_DIR="${OUTPUT_DIR:-/var/backups/sitecraft}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"

if [[ ! -f "$ENV_FILE" ]]; then
    echo "missing env file: $ENV_FILE"
    exit 1
fi

set -a
source "$ENV_FILE"
set +a

if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "DATABASE_URL is required"
    exit 1
fi

mkdir -p "$OUTPUT_DIR"
pg_dump "$DATABASE_URL" | gzip -c > "$OUTPUT_DIR/sitecraft_db_${TIMESTAMP}.sql.gz"
