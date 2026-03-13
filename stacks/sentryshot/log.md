# SentryShot

## Sources
- Installation docs: https://codeberg.org/SentryShot/sentryshot/src/branch/master/docs/1_Installation.md
- Container registry: codeberg.org/sentryshot/sentryshot
- Source: https://codeberg.org/SentryShot/sentryshot

## Notes
- Compose converted from docker run command in official installation docs
- Image hosted on Codeberg container registry (not Docker Hub)
- Port 2020 for web UI (live view at /live)
- Config and storage directories bind-mounted
- Must enable an auth plugin in configs before first use (e.g. auth_none)
- TZ environment variable for timezone
