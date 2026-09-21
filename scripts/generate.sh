#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

if command -v npm >/dev/null 2>&1 && [ -d "${ROOT_DIR}/web" ]; then
    echo "==> Generating TypeScript TanStack Query hooks via Orval..."
    (cd "${ROOT_DIR}/web" && npm run generate)
    echo "==> Orval generation complete."
fi
