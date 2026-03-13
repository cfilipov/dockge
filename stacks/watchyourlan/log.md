# WatchYourLAN

## Status: DONE

## Sources
- https://raw.githubusercontent.com/aceberg/WatchYourLAN/main/docker-compose.yml — official compose file

## Notes
- Image: `aceberg/watchyourlan` (also on GHCR: `ghcr.io/aceberg/watchyourlan`)
- Uses host networking for ARP scanning
- Requires TZ and IFACES environment variables
- Default port 8840
- Changed bind mount to named volume for portability
- Set IFACES to "eth0" as placeholder (user must configure)
