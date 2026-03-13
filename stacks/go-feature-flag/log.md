# GO Feature Flag

## Source
- Repository: https://github.com/thomaspoignant/go-feature-flag
- Docker run command from README: `docker run -p 1031:1031 -v $(pwd)/flag-config.yaml:/goff/flag-config.yaml -v $(pwd)/goff-proxy.yaml:/goff/goff-proxy.yaml gofeatureflag/go-feature-flag:latest`
- Configuration docs: https://gofeatureflag.org/docs/relay-proxy/configure-relay-proxy

## Notes
- No docker-compose file in the repository; compose derived from the docker run command in the README
- Image: `gofeatureflag/go-feature-flag:latest` from Docker Hub
- Port 1031: HTTP API (relay proxy)
- Requires two config files mounted into the container:
  - `goff-proxy.yaml` — relay proxy configuration (retriever setup)
  - `flags.yaml` — feature flag definitions
- Both config files are included as minimal working examples
