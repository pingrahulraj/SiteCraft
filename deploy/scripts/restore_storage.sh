#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "usage: restore_storage.sh <backup.tar.gz>"
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

if [[ -z "${STORAGE_ROOT:-}" ]]; then
    echo "STORAGE_ROOT is required"
    exit 1
fi

mkdir -p "$(dirname "$STORAGE_ROOT")"
tar -xzf "$BACKUP_FILE" -C "$(dirname "$STORAGE_ROOT")"
