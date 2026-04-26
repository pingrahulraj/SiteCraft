#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="${ENV_FILE:-/etc/sitecraft/sitecraft.env}"

set -a
source "$ENV_FILE"
set +a

cd "$ROOT_DIR"
python3 run_project.py migrate
