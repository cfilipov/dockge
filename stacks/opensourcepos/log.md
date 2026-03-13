# Open Source POS Stack

## Research
- Found docker-compose.yml in opensourcepos/opensourcepos repo
- Original uses `include:` to reference docker/docker-mysql.yml
- Merged both files into a single compose.yaml
- Removed `sqlscript` init container (used `volumes_from` which is deprecated)
- Services: ospos (PHP app), MariaDB 10.5

## Changes from upstream
- Merged docker-mysql.yml include into single compose.yaml
- Removed sqlscript service (init container for SQL seeding)
- Removed `volumes_from` (deprecated in compose v3)
- No ${VAR} substitution needed — all values inline
