# FitTrackee

## Sources
- Repository: https://github.com/SamR1/FitTrackee
- Compose file: `docker-compose.yml` from master branch
- Env file: `.env.docker.example` from master branch

## Notes
- Image: `fittrackee/fittrackee:v1.1.2` (Docker Hub)
- Database: PostGIS 17-3.5 (PostgreSQL with geospatial extensions)
- Minimal setup (2 services): app + database
- Optional Redis and workers services available for multi-user (commented in source)
- Removed `post_start` directive (requires Docker Compose 2.30+)
- Uses internal/external network separation
