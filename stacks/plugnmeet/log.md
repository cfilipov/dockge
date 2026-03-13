# plugNmeet

## Source
- Repository: https://github.com/mynaparrot/plugNmeet-server
- docker-compose_sample.yaml: https://raw.githubusercontent.com/mynaparrot/plugNmeet-server/main/docker-compose_sample.yaml

## Images
- mynaparrot/plugnmeet-server:latest
- livekit/livekit-server:latest
- redis:7-alpine
- mariadb:11
- nats:2-alpine
- mynaparrot/plugnmeet-etherpad:latest

## Notes
- Original sample compose uses `build` for plugnmeet-api; converted to use published Docker Hub image
- Requires config.yaml and livekit.yaml config files (not included; see config_sample.yaml in repo)
- Original also includes livekit-ingress and livekit-sip services (omitted as optional)
- MariaDB data persisted via named volume
- Full installation guide at https://www.plugnmeet.org/docs/installation
