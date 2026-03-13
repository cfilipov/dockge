# LedgerSMB Stack Research Log

## Sources
- Main repo: https://github.com/ledgersmb/LedgerSMB
- Docker repo: https://github.com/ledgersmb/ledgersmb-docker
- Reference compose: https://raw.githubusercontent.com/ledgersmb/ledgersmb-docker/1.13/docker-compose.yml

## Notes
- Image: `ghcr.io/ledgersmb/ledgersmb:1.13` (GitHub Container Registry)
- Database: PostgreSQL 15 (Alpine)
- Access setup at http://localhost:5762/setup.pl, login at http://localhost:5762/login.pl
- Removed port 80 mapping (redundant with 5762), kept only 5762
- Removed `version: "3.2"` field for Compose V2 compatibility
- LSMB_WORKERS controls HTTP worker processes (default 5)
- All data persisted in PostgreSQL via pgdata volume
- Internal network isolates postgres from external access
