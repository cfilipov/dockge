# Kill Bill

- **Source**: https://github.com/killbill/killbill
- **Compose reference**: `killbill-cloud` repo `docker/compose/docker-compose.kb.yml`
- **Status**: ok
- **Services**: killbill (billing engine), kaui (admin UI), db (MariaDB with Kill Bill schema)
- **Notes**: Based on the official compose from killbill-cloud repository. Uses Kill Bill's custom MariaDB image (0.24) which includes pre-loaded schemas for both killbill and kaui databases.
