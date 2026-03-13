# Aimeos Stack

## Research
- Found docker-compose.yml in the aimeos/aimeos GitHub repo (Laravel Sail based)
- Original uses `build:` for the app service; replaced with `image: aimeos/aimeos:latest`
- Services: app (Laravel), MySQL 8.0, MinIO (S3-compatible storage), Mailhog (mail testing)
- All ${VAR} references have defaults and are defined in .env

## Changes from upstream
- Replaced `build:` context with `image: aimeos/aimeos:latest`
- Removed bind-mount volumes that reference local source code
- Added named volume for app data
- Kept MySQL, MinIO, and Mailhog services as-is with real images
