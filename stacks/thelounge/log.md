# The Lounge

## Sources
- Docker repo: https://github.com/thelounge/thelounge-docker
- Official docker-compose.yml: https://github.com/thelounge/thelounge-docker/blob/master/docker-compose.yml
- Image: ghcr.io/thelounge/thelounge:latest

## Notes
- Taken directly from official docker-compose.yml
- Port 9000 for web interface
- Data volume at /var/opt/thelounge
- Add users via: docker exec -it thelounge thelounge add <username>
