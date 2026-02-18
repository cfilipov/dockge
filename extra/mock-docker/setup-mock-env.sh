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

# blog and web-app are running; monitoring, test-alpine, database are exited
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
echo

# --- 3. Seed image update cache in SQLite ---
if [[ -f "$DB_PATH" ]]; then
    echo "Seeding image_update_cache in database..."

    # Use Node.js with the project's SQLite module to seed data
    node -e "
const Database = require('$(find "$REPO_DIR/node_modules" -path "*/@louislam/sqlite3/lib/sqlite3.js" -print -quit)');

const db = new Database.Database('$DB_PATH');

db.serialize(() => {
    // Ensure table exists
    db.run(\`CREATE TABLE IF NOT EXISTS image_update_cache (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        stack_name TEXT NOT NULL,
        service_name TEXT NOT NULL,
        image_reference TEXT,
        local_digest TEXT,
        remote_digest TEXT,
        has_update INTEGER DEFAULT 0,
        last_checked INTEGER,
        UNIQUE(stack_name, service_name)
    )\`);

    // Clear existing data
    db.run('DELETE FROM image_update_cache');

    const now = Math.floor(Date.now() / 1000);
    const entries = [
        // web-app: nginx has update, redis does not
        ['web-app', 'nginx', 'nginx:latest', 'sha256:aaa', 'sha256:bbb', 1, now],
        ['web-app', 'redis', 'redis:alpine', 'sha256:ccc', 'sha256:ccc', 0, now],
        // blog: wordpress has update, mysql does not
        ['blog', 'wordpress', 'wordpress:6.4', 'sha256:ddd', 'sha256:eee', 1, now],
        ['blog', 'mysql', 'mysql:8.0', 'sha256:fff', 'sha256:fff', 0, now],
        // monitoring: no updates
        ['monitoring', 'grafana', 'grafana/grafana:latest', 'sha256:ggg', 'sha256:ggg', 0, now],
        // test-alpine: no updates
        ['test-alpine', 'alpine', 'alpine:latest', 'sha256:hhh', 'sha256:hhh', 0, now],
        // database: has update
        ['database', 'postgres', 'postgres:16', 'sha256:iii', 'sha256:jjj', 1, now],
    ];

    const stmt = db.prepare(
        'INSERT OR REPLACE INTO image_update_cache (stack_name, service_name, image_reference, local_digest, remote_digest, has_update, last_checked) VALUES (?, ?, ?, ?, ?, ?, ?)'
    );

    for (const entry of entries) {
        stmt.run(...entry);
    }
    stmt.finalize();

    console.log('  Inserted ' + entries.length + ' image update cache entries');
});

db.close();
" 2>&1

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
