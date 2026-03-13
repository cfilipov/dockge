# kiwix-serve

## Source
- GitHub: https://github.com/kiwix/kiwix-tools
- GHCR: ghcr.io/kiwix/kiwix-serve

## Research
- Dedicated kiwix-serve Docker image on GHCR
- Serves ZIM files (offline Wikipedia, etc.)
- Default port 8080, command takes ZIM file paths as arguments
- Multi-arch support (amd64, arm64, arm/v6, arm/v7)

## Compose
- Image: ghcr.io/kiwix/kiwix-serve:3.8.2
- Port: 8080 (Web UI)
- Volume: data directory for ZIM files
- Command: serves all .zim files in /data
