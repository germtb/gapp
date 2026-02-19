#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

gap codegen \
  --proto "$SCRIPT_DIR/proto/service.proto" \
  --go-out "$SCRIPT_DIR/server/generated" \
  --ts-out "$SCRIPT_DIR/client/src/generated" \
  --routes-dir "$SCRIPT_DIR/client/src/routes" \
  --preload-out "$SCRIPT_DIR/server/generated/preload_routes.go" \
  --force
