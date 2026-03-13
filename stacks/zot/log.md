# ZOT OCI Registry

## Sources
- Repository: https://github.com/project-zot/zot
- Dockerfile: https://github.com/project-zot/zot/blob/main/build/Dockerfile-minimal
- Minimal config example: https://github.com/project-zot/zot/blob/main/examples/config-minimal.json
- Docs: https://zotregistry.dev

## Notes
- Image: `ghcr.io/project-zot/zot-minimal-linux-amd64:latest` (from GHCR, per Dockerfile multiarch build)
- Port 5000 (from Dockerfile EXPOSE)
- Config at /etc/zot/config.json (from Dockerfile CMD)
- Storage at /var/lib/registry (from Dockerfile default config)
- OCI-native image registry, no database required
- Config file derived from examples/config-minimal.json with address changed to 0.0.0.0
