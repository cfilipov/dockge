# Spree Commerce

- **Source**: https://github.com/spree/spree
- **Status**: skipped
- **Reason**: The project has a docker-compose.yml but all app services use `build:` directives with YAML anchors. The only Docker Hub image (spreecommerce/spree) was last updated in 2019 and is severely outdated (Spree 3.x vs current 5.x). No current published Docker image exists for Spree.
