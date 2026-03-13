# SANE Network Scanning

- **Status**: created
- **Source**: https://github.com/sbs20/scanservjs
- **Image**: sbs20/scanservjs:latest (Docker Hub, 600k+ pulls)
- **Notes**: scanservjs is the most popular Docker-based SANE scanner web UI. The sane-project/sane-backends repo itself does not provide Docker images. This uses scanservjs which wraps SANE with a Node.js web interface. Requires USB passthrough for local scanners or SANED_NET_HOSTS for network scanners.
