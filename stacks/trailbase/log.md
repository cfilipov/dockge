# TrailBase — Research Log

## Sources checked
1. GitHub repo `trailbaseio/trailbase` README — found `docker run` command with image `trailbase/trailbase`

## Compose file origin
Converted from the `docker run` alias in the official GitHub README.

## Modifications
- Converted `docker run` flags to Compose V2 format
- Changed bind mount to relative path (`./traildepot`)
- Added `restart: unless-stopped`
