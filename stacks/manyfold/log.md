# Manyfold Stack Research Log

## Sources
- Official example: https://github.com/manyfold3d/manyfold/blob/main/docker-compose.example.yml
- GitHub: https://github.com/manyfold3d/manyfold
- Image: ghcr.io/manyfold3d/manyfold:latest

## Notes
- Manyfold is a self-hosted 3D model manager for 3D printing collections
- Compose based directly on the official docker-compose.example.yml from the repository
- Requires PostgreSQL, Redis, and the app service
- Removed `version: "3"` and `links:` (deprecated in Compose V2)
- Added .env file for SECRET_KEY_BASE and DATABASE_PASSWORD
- Models directory bind-mounted at ./models (uncommented from the example)
- Port 3214 is the default web UI port
