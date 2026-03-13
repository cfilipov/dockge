# TrackWatch

- **Source**: https://github.com/emlopezr/trackwatch
- **Image**: ghcr.io/emlopezr/trackwatch:latest
- **Port**: 80
- **Description**: Self-hosted Spotify release tracker. Auto-syncs new releases from followed artists to a Spotify playlist. Includes discography generator and ghost track cleaner.
- **Notes**: All-in-one image bundles PostgreSQL internally. Requires Spotify Developer app credentials. Redirect URI: http://127.0.0.1:80/callback (Spotify requires 127.0.0.1, not localhost).
