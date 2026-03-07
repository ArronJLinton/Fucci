#!/usr/bin/env bash
# Deploy the API to Fly.io. DB migrations run automatically via release_command in fly.toml.
#
# Usage:
#   ./scripts/deploy-api.sh              # Deploy (migrations run on Fly during release)
#   RUN_MIGRATIONS=1 ./scripts/deploy-api.sh   # Run migrations locally first (uses DB_URL), then deploy
#
# Prerequisites: flyctl installed and logged in; DB_URL set (for RUN_MIGRATIONS=1).

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

if [[ -n "${RUN_MIGRATIONS:-}" ]]; then
  echo "Running migrations locally (DB_URL) before deploy..."
  yarn migrate
  echo "Migrations complete."
fi

echo "Deploying API to Fly.io (migrations will run on release)..."
cd services/api && flyctl deploy --remote-only
