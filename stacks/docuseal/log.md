# DocuSeal

## What was done
- Based on official docker-compose.yml from docusealco/docuseal
- Removed Caddy reverse proxy (not needed for local testing)
- Services: app (DocuSeal), PostgreSQL
- Images: docuseal/docuseal:latest, postgres:16
- Port: 3000 (web UI)
- No .env needed - no variable substitution used
