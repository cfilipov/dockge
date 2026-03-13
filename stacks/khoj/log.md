# Khoj

## Sources
- Repository: https://github.com/khoj-ai/khoj
- Compose file: https://github.com/khoj-ai/khoj/blob/master/docker-compose.yml

## Notes
- Based on the official docker-compose.yml from the repository root
- Includes server, PostgreSQL with pgvector, SearXNG (search), and Terrarium (sandbox)
- Removed the optional `computer` service (VNC-based) to keep it simpler
- Removed duplicate volume mount (khoj_models was mounted to two paths; kept one)
- Access at http://localhost:42110
- Default admin: username@example.com / password
