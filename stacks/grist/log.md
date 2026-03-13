# Grist

## Sources
- Official README Docker instructions: https://github.com/gristlabs/grist-core/blob/main/README.md
- Docker Hub image: `gristlabs/grist`
- No docker-compose.yml in repo; composed from documented docker run commands

## Notes
- Port 8484 is the default
- Data persisted to /persist volume
- `gristlabs/grist` includes enterprise extensions (inactive by default); `gristlabs/grist-oss` is the pure OSS variant
- Default user is you@example.com; configure GRIST_DEFAULT_EMAIL to change
- For production, configure SSO via SAML/OIDC
