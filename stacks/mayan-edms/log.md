# Mayan EDMS

## What was done
- Based on official docker-compose.yml from mayan-edms/Mayan-EDMS repo
- Simplified: removed profiles, traefik, elasticsearch, extra workers (all optional)
- Kept all_in_one "app" service as primary
- Services: app (Mayan EDMS), PostgreSQL, Redis, RabbitMQ
- Images: mayanedms/mayanedms:s4.7, postgres:14-alpine, redis:7-alpine, rabbitmq:3-management-alpine
- Port: 80 -> 8000
- All credentials extracted to .env
