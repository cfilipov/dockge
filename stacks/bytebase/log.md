# Bytebase Stack

- Used official `bytebase/bytebase` image from Docker Hub
- Port 8080 exposed for web UI
- Data persisted to named volume
- `init: true` for proper signal handling (as per official docs)
- No .env needed - no variable substitution used
