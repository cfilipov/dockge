# Digibuzzer

- **Source**: https://codeberg.org/ladigitale/digibuzzer
- **Description**: Virtual game room around a connected buzzer (documentation in French)
- **Demo**: https://digibuzzer.app/
- **License**: AGPL-3.0
- **Language**: Node.js
- **Image**: node:20-alpine (no official Docker image found)
- **Notes**: No official Docker image exists on Docker Hub or Codeberg container registry. This stack uses a generic Node.js image. In production, you would clone the source and mount the app directory. The Codeberg repo had no Dockerfile or docker-compose.yml at the expected paths.
