# EverShop Stack

## Research
- Found docker-compose.yml in evershopcommerce/evershop GitHub repo
- Already uses `image: evershop/evershop:latest` (no build step)
- Services: app (Node.js e-commerce), PostgreSQL 16
- No ${VAR} substitution needed — all values are inline

## Changes from upstream
- Minimal changes — upstream compose was already image-based
- Renamed network for consistency
