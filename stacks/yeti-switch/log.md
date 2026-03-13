# Yeti-Switch Stack

## Research
- GitHub org: yeti-switch (yeti-web, sems-yeti, sems)
- Docker Hub image: switchyeti/yeti-web (tags: pg13, pg12, pg11)
- Yeti is an open-source SIP SBC (Session Border Controller) with billing
- Architecture: yeti-web (Ruby admin UI) + SEMS (SIP engine) + PostgreSQL (routing + CDR)
- No official docker-compose; only a dev Dockerfile for PostgreSQL

## Compose
- Uses `switchyeti/yeti-web:pg13` for the admin web interface
- Two PostgreSQL 13 instances: one for routing config, one for CDRs
- Web UI exposed on port 3000
- DB credentials in .env file
- Note: Full production setup would also need SEMS (no Docker image available)
