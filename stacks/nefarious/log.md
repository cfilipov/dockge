# nefarious

- **Source**: https://github.com/lardbit/nefarious
- **Image**: `lardbit/nefarious:latest`
- **Description**: Web application that automatically downloads Movies and TV Shows using Jackett for torrent search and Transmission for downloading. Multi-service stack.
- **Ports**: 8080 (nefarious), 9117 (jackett), 9091 (transmission)
- **Services**: nefarious, redis, jackett, transmission
- **Volumes**: jackett-config, transmission-config, downloads
- **Key env vars**: NEFARIOUS_USER, NEFARIOUS_PASS
- **Category**: Media Management
