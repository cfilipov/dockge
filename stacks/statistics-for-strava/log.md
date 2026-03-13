# Statistics for Strava

- **Source**: https://github.com/robiningelbrecht/strava-statistics
- **Status**: ok
- **Image**: robiningelbrecht/strava-statistics:latest (Docker Hub)
- **Notes**: Based on upstream docker-compose.yml (master branch). Replaced build directives with the Docker Hub image. App serves web UI on port 8080, daemon runs background sync. Requires Strava API credentials in config/app directory. Created empty config/app directory for bind mount.
