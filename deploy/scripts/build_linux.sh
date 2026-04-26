#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR/backend"

mkdir -p "$ROOT_DIR/bin"
GOOS=linux GOARCH=amd64 go build -o "$ROOT_DIR/bin/sitecraft-api" ./cmd/api
GOOS=linux GOARCH=amd64 go build -o "$ROOT_DIR/bin/sitecraft-delivery" ./cmd/delivery

cd "$ROOT_DIR/frontend"
npm run build
