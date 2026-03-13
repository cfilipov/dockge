# Upsnap

## Status: DONE

## Sources
- https://raw.githubusercontent.com/seriousm4x/UpSnap/master/docker-compose.yml — official compose file

## Notes
- Image: `ghcr.io/seriousm4x/upsnap:5` (also available as `seriousm4x/upsnap:5` on Docker Hub)
- Uses host networking for Wake-on-LAN and network scanning
- Data stored in ./data bind mount
- Optional env vars for scan range, interval, timezone
