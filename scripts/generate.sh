#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "==> Generating Go server interfaces and models from api/openapi.yaml..."
mkdir -p "${ROOT_DIR}/internal/adapters/in/http"

# Use go run directly (no PATH or asdf shim dependencies needed)
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest \
  -config "${ROOT_DIR}/api/oapi-codegen.yaml" \
  "${ROOT_DIR}/api/openapi.yaml"

echo "==> Go generation complete."

if command -v npm >/dev/null 2>&1 && [ -d "${ROOT_DIR}/web" ]; then
    echo "==> Generating TypeScript TanStack Query hooks via Orval..."
    (cd "${ROOT_DIR}/web" && npx orval)
    echo "==> Orval generation complete."
fi
