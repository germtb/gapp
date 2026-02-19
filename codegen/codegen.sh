#!/bin/bash
# Gap framework codegen script
#
# Usage: ./codegen.sh --proto <proto_file> --go-out <dir> --ts-out <dir> [--routes-dir <dir>] [--preload-out <path>]
#
# This script:
# 1. Runs protoc to generate Go and TypeScript code from proto files
# 2. Optionally generates preload route configuration from TypeScript route files

set -euo pipefail

PROTO_FILE=""
GO_OUT=""
TS_OUT=""
ROUTES_DIR=""
PRELOAD_OUT=""
GO_PACKAGE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --proto) PROTO_FILE="$2"; shift 2 ;;
    --go-out) GO_OUT="$2"; shift 2 ;;
    --ts-out) TS_OUT="$2"; shift 2 ;;
    --routes-dir) ROUTES_DIR="$2"; shift 2 ;;
    --preload-out) PRELOAD_OUT="$2"; shift 2 ;;
    --go-package) GO_PACKAGE="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ -z "$PROTO_FILE" ]]; then
  echo "Error: --proto is required"
  exit 1
fi

PROTO_DIR=$(dirname "$PROTO_FILE")
PROTO_NAME=$(basename "$PROTO_FILE")

# Generate Go code
if [[ -n "$GO_OUT" ]]; then
  echo "Generating Go code..."
  mkdir -p "$GO_OUT"

  GO_OPT="paths=source_relative"
  if [[ -n "$GO_PACKAGE" ]]; then
    GO_OPT="${GO_OPT},M${PROTO_NAME}=${GO_PACKAGE}"
  fi

  protoc \
    --proto_path="$PROTO_DIR" \
    --go_out="$GO_OUT" \
    --go_opt="$GO_OPT" \
    "$PROTO_NAME"

  echo "  Go code generated in $GO_OUT"
fi

# Generate TypeScript code
if [[ -n "$TS_OUT" ]]; then
  echo "Generating TypeScript code..."
  mkdir -p "$TS_OUT"

  protoc \
    --proto_path="$PROTO_DIR" \
    --plugin=protoc-gen-ts_proto="$(which protoc-gen-ts_proto 2>/dev/null || echo ./node_modules/.bin/protoc-gen-ts_proto)" \
    --ts_proto_out="$TS_OUT" \
    --ts_proto_opt=outputServices=default \
    --ts_proto_opt=esModuleInterop=true \
    --ts_proto_opt=useOptionals=messages \
    "$PROTO_NAME"

  echo "  TypeScript code generated in $TS_OUT"
fi

# Generate preload routes config
if [[ -n "$ROUTES_DIR" && -n "$PRELOAD_OUT" ]]; then
  echo "Generating preload route configuration..."

  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  bun run "$SCRIPT_DIR/generate-preload-config.ts" \
    --routes-dir "$ROUTES_DIR" \
    --output "$PRELOAD_OUT"
fi

echo "Done."
