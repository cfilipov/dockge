# Evergreen ILS - Research Log

## Sources
1. **GitHub repo (main)**: https://github.com/evergreen-library-system/Evergreen - No Docker files in the main repo
2. **Docker Hub**: https://hub.docker.com/r/mobiusoffice/evergreen-ils - Official community image (23k+ pulls, 8 stars)
3. **Docker source repo**: https://github.com/mcoia/eg-docker - Contains Dockerfile and docker-compose.yml
4. **docker-compose.yml**: https://github.com/mcoia/eg-docker/blob/master/generic-dockerhub/docker-compose.yml
5. **.env**: https://github.com/mcoia/eg-docker/blob/master/generic-dockerhub/.env

## Notes
- The upstream compose uses `build:` context; adapted to use the published Docker Hub image `mobiusoffice/evergreen-ils`
- All-in-one container: includes Ubuntu, PostgreSQL, Apache, Evergreen, and OpenSRF
- Default credentials: admin / demo123
- Hostname must be set (`-h` flag) or OpenSRF fails to start
- Original compose had a bind mount to `/mnt/evergreen`; replaced with a named volume
- Ports: HTTP (80), HTTPS (443), Z39.50 (210), SIP (6001), SSH (22), PostgreSQL (5432)
