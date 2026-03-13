# Lila (Lichess)

- **Source**: https://github.com/lichess-org/lila
- **Docker**: https://github.com/lichess-org/lila-docker
- **Description**: Free, open-source chess server powering lichess.org
- **Image**: ghcr.io/lichess-org/lila-docker:main (mono image for quick setup)
- **Dependencies**: MongoDB 8.x (replica set), Redis 8.x
- **Notes**: The full Lichess development setup uses many more services (lila-ws, fishnet, stockfish, etc.). This stack uses the simplified "mono" profile image from the official lila-docker repository that bundles the core application. MongoDB requires replica set mode for change streams.
