# BookLogr

## Source
- GitHub: https://github.com/Mozzo1000/booklogr
- Docker Hub: mozzo/booklogr, mozzo/booklogr-web

## Research
- Found docker-compose.yml in the repository
- Two services: API (Python/Flask) and Web (frontend)
- SQLite by default, PostgreSQL optional
- Upstream compose used localhost for API endpoint; changed to service name for inter-container communication

## Compose
- Images: mozzo/booklogr:v1.7.0, mozzo/booklogr-web:v1.7.0
- Ports: 5000 (API), 5150 (Web UI)
- .env file for AUTH_SECRET_KEY variable substitution
