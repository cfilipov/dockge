# Open QuarterMaster

## Source
- Repository: https://github.com/Epic-Breakfast-Productions/OpenQuarterMaster
- Docker images: ebprod/oqm-core-api, ebprod/oqm-core-base_station

## Compose Source
- Fetched from: https://raw.githubusercontent.com/Epic-Breakfast-Productions/OpenQuarterMaster/main/deployment/Compose/compose/compose.yaml
- .env from: https://raw.githubusercontent.com/Epic-Breakfast-Productions/OpenQuarterMaster/main/deployment/Compose/compose/.env.example

## Changes from Original
- Removed `version: '3.8'` field (Compose V2)
- Replaced bind-mount data dirs with named volumes
- Removed Keycloak realm file volume mount (requires separate setup)
- Added default values to all variable substitutions
- Replaced placeholder client secret with "changeme"

## Notes
- Full setup requires importing a Keycloak realm JSON file (available in repo at deployment/Compose/compose/setup/oqm-realm.json)
- core-base-station uses network_mode: host (as per upstream)

## Services
- **mongo**: MongoDB 6 (inventory data)
- **postgres**: PostgreSQL 18 (Keycloak auth data)
- **keycloak**: Identity/access management (port 8100)
- **core-api**: OQM Core API (port 8101)
- **core-base-station**: Web UI frontend (port 8102)
