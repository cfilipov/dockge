# Beaver Habit Tracker

## Source
- GitHub: https://github.com/daya0576/beaverhabits
- Docker Hub: https://hub.docker.com/r/daya0576/beaverhabits

## Research
- Docker Compose example found in project README
- Image: `daya0576/beaverhabits:latest`
- Supports `USER_DISK` (JSON file) and `DATABASE` (SQLite) storage modes
- Removed `TRUSTED_LOCAL_EMAIL` default to avoid auth bypass in production
- Removed `INDEX_HABIT_DATE_COLUMNS` as it's cosmetic customization
