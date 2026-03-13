# Atheos

## Sources
- GitHub: https://github.com/Atheos/Atheos
- Docker Hub: https://hub.docker.com/r/hlsiira/atheos (26k+ pulls, last updated Dec 2025)
- Docker Hub API: https://hub.docker.com/v2/repositories/hlsiira/atheos/

## Research Notes
- No docker-compose file in the repository
- No Dockerfile in the repository root
- Docker Hub image `hlsiira/atheos` exists with a Dockerfile based on Ubuntu + PHP 7.4 + Apache2
- Exposes ports 80 and 443
- Volumes: /var/www/html, /etc/apache2
- Compose file created based on Docker Hub image info and Dockerfile analysis
