# Feeds Fun

## Source
https://github.com/Tiendil/feeds.fun

## Notes
- News reader with tags-based filtering and scoring
- Uses PostgreSQL for data storage
- Dev compose uses custom build images; simplified for self-hosting with tiendil/feeds.fun
- Original compose has Caddy proxy, Keycloak IdP, OAuth2 proxy for multi-user mode
- Simplified to single backend + PostgreSQL for testing
