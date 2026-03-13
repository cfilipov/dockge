# opencanary (Canary Tokens)

## Status: DONE

## Sources
- https://raw.githubusercontent.com/thinkst/opencanary/master/docker-compose.yml — official compose file
- https://raw.githubusercontent.com/thinkst/opencanary/master/data/.opencanary.conf — default config

## Notes
- Image: `thinkst/opencanary`
- Uses `network_mode: host` for honeypot functionality
- Config file bind-mounted from `./data/.opencanary.conf`
- Simplified compose from upstream (removed build directives and YAML anchors)
- Only FTP and HTTP enabled by default; other services can be enabled in config
