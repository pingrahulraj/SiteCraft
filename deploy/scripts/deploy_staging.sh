#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TARGET_DIR="${TARGET_DIR:-/opt/sitecraft}"

"$ROOT_DIR/deploy/scripts/build_linux.sh"

rsync -av --delete "$ROOT_DIR/backend/" "$TARGET_DIR/backend/"
rsync -av --delete "$ROOT_DIR/frontend/dist/" "$TARGET_DIR/frontend/dist/"
rsync -av "$ROOT_DIR/bin/sitecraft-api" "$TARGET_DIR/bin/sitecraft-api"

sudo systemctl daemon-reload
sudo systemctl restart websitebuilder-api.service
sudo systemctl restart websitebuilder-delivery.service
