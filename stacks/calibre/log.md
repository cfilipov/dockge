# Calibre

## Source
- LinuxServer.io: https://github.com/linuxserver/docker-calibre
- Docker Hub: linuxserver/calibre

## Research
- Found complete compose example in linuxserver/docker-calibre README
- Desktop GUI via web browser (KasmVNC)
- Three ports: 8080 (GUI), 8181 (HTTPS GUI), 8081 (webserver)
- Requires seccomp:unconfined and shm_size

## Compose
- Image: lscr.io/linuxserver/calibre:latest
- Ports: 8080 (GUI), 8181 (HTTPS), 8081 (content server)
- Volumes: config directory
