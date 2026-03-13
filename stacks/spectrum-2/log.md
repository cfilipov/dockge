# Spectrum 2

- **Status**: ok
- **Source**: https://github.com/SpectrumIM/spectrum2
- **Based on**: Official docker-compose.yml from master branch
- **Images**: spectrum2/spectrum:master, spectrum2/prosody:latest, spectrum2/nginx:latest
- **Notes**: XMPP transport/gateway. Connects to Prosody XMPP server. Replaced bind-mount config paths with generic volume mounts.
