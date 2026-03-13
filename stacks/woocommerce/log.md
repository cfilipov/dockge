# WooCommerce Stack

## Research
- WooCommerce is a WordPress plugin, no standalone Docker image
- Standard approach: WordPress image + MySQL + WP-CLI for plugin install

## Compose
- Used official `wordpress:latest` image
- MySQL 8.0 for database
- Added `wpcli` service (wordpress:cli) in `setup` profile to install WooCommerce plugin
- Setup service runs once via `docker compose --profile setup run wpcli`
