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

if [[ -z "${STORAGE_ROOT:-}" ]]; then
    echo "STORAGE_ROOT is required"
    exit 1
fi
if [[ ! -d "$STORAGE_ROOT" ]]; then
    echo "storage root not found: $STORAGE_ROOT"
    exit 1
fi

mkdir -p "$OUTPUT_DIR"
tar -czf "$OUTPUT_DIR/sitecraft_storage_${TIMESTAMP}.tar.gz" -C "$(dirname "$STORAGE_ROOT")" "$(basename "$STORAGE_ROOT")"
