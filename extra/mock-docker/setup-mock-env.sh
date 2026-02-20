#!/usr/bin/env bash
# Set up the mock development environment for Dockge.
#
# This script:
# 1. Copies test stacks to DOCKGE_STACKS_DIR (default: /opt/stacks)
# 2. Seeds mock Docker state (running/exited) in /tmp/mock-docker/state
# 3. Seeds the SQLite database with sample image update cache entries
#
# Usage:
#   ./extra/mock-docker/setup-mock-env.sh
#
# Prerequisites:
#   - /opt/stacks (or $DOCKGE_STACKS_DIR) must exist and be writable
#   - Run `pnpm install` first so the SQLite module is available

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
STACKS_DIR="${DOCKGE_STACKS_DIR:-/opt/stacks}"
STATE_DIR="/tmp/mock-docker/state"
DB_PATH="$REPO_DIR/data/dockge.db"

echo "Setting up mock development environment..."
echo "  Repo:   $REPO_DIR"
echo "  Stacks: $STACKS_DIR"
echo "  State:  $STATE_DIR"
echo "  DB:     $DB_PATH"
echo

# --- 1. Copy test stacks ---
echo "Copying test stacks to $STACKS_DIR..."
for stack_dir in "$REPO_DIR/extra/test-stacks"/*/; do
    name=$(basename "$stack_dir")
    mkdir -p "$STACKS_DIR/$name"
    cp "$stack_dir/compose.yaml" "$STACKS_DIR/$name/compose.yaml"
    echo "  $name"
done
echo

# --- 2. Seed mock Docker state ---
echo "Seeding mock Docker state..."
mkdir -p "$STATE_DIR"

# blog and web-app are running; monitoring, test-alpine, database are exited; cache is down (no state)
for name in blog web-app; do
    mkdir -p "$STATE_DIR/$name"
    echo "running" > "$STATE_DIR/$name/status"
    echo "  $name → running"
done

for name in monitoring test-alpine database; do
    mkdir -p "$STATE_DIR/$name"
    echo "exited" > "$STATE_DIR/$name/status"
    echo "  $name → exited"
done

# cache has no state file — it appears as "down" (compose.yaml exists but never started)
rm -rf "$STATE_DIR/cache"
echo "  cache → down (no state)"
echo

# --- 3. Seed image update cache in SQLite ---
if [[ -f "$DB_PATH" ]]; then
    echo "Seeding image_update_cache in database..."

    sqlite3 "$DB_PATH" <<'SQL'
DELETE FROM image_update_cache;
INSERT OR REPLACE INTO image_update_cache
    (stack_name, service_name, image_reference, local_digest, remote_digest, has_update, last_checked)
VALUES
    ('web-app', 'nginx', 'nginx:latest', 'sha256:aaa', 'sha256:bbb', 1, strftime('%s','now')),
    ('web-app', 'redis', 'redis:alpine', 'sha256:ccc', 'sha256:ccc', 0, strftime('%s','now')),
    ('blog', 'wordpress', 'wordpress:6.4', 'sha256:ddd', 'sha256:eee', 1, strftime('%s','now')),
    ('blog', 'mysql', 'mysql:8.0', 'sha256:fff', 'sha256:fff', 0, strftime('%s','now')),
    ('monitoring', 'grafana', 'grafana/grafana:latest', 'sha256:ggg', 'sha256:ggg', 0, strftime('%s','now')),
    ('test-alpine', 'alpine', 'alpine:latest', 'sha256:hhh', 'sha256:hhh', 0, strftime('%s','now')),
    ('database', 'postgres', 'postgres:16', 'sha256:iii', 'sha256:jjj', 1, strftime('%s','now'));
SQL

    echo "  Inserted $(sqlite3 "$DB_PATH" 'SELECT COUNT(*) FROM image_update_cache') entries"
    echo
else
    echo "Database not found at $DB_PATH — skipping cache seeding."
    echo "Run the dev server once first to create the database, then re-run this script."
    echo
fi

echo "Done! To use the mock Docker CLI:"
echo "  export PATH=\"$REPO_DIR/extra/mock-docker:\$PATH\""
echo "  export DOCKGE_STACKS_DIR=$STACKS_DIR"
echo "  pnpm run dev"
