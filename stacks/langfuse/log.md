# Langfuse

## Sources
- GitHub: https://github.com/langfuse/langfuse
- Official docker-compose.yml: https://github.com/langfuse/langfuse/blob/main/docker-compose.yml

## Research Notes
- Official docker-compose.yml found in repository root
- Multi-service stack: langfuse-web (port 3000), langfuse-worker (port 3030), ClickHouse, MinIO, Redis, PostgreSQL
- Images: langfuse/langfuse:3 (web), langfuse/langfuse-worker:3 (worker)
- Uses YAML anchors for shared environment config
- Requires multiple credentials (SALT, ENCRYPTION_KEY, NEXTAUTH_SECRET, DB passwords)
- Compose file adapted from official with minor cleanup (removed some optional env vars)
