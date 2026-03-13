# FreshRSS

## Source
https://github.com/FreshRSS/FreshRSS

## Notes
- Self-hosted RSS feed aggregator, Google Reader alternative
- Official Docker image: freshrss/freshrss
- Compose based on upstream docker-compose.yml from the edge branch
- Built-in cron for feed refresh (CRON_MIN=3,33 means every 30 min)
- Supports extensions via volume mount
- Lightweight with SQLite by default; supports PostgreSQL/MySQL
