# ZNC

## Sources
- Docker repo: https://github.com/znc/znc-docker
- Docker Hub official image: https://hub.docker.com/_/znc
- Docker library docs: https://github.com/docker-library/docs/blob/master/znc/content.md
- Image: znc (official Docker Hub image)

## Notes
- Official Docker Hub image
- First run requires: docker compose run --rm znc --makeconf (interactive setup)
- Port is configured during --makeconf (6501 used as example; 6667/6697 not recommended as browsers block them)
- Data volume at /znc-data stores config and modules
