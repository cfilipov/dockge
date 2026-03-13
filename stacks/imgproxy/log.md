# imgproxy

## Source
- GitHub: https://github.com/imgproxy/imgproxy
- Docker image: `ghcr.io/imgproxy/imgproxy:latest`

## Research
- README provides `docker run -p 8080:8080 -it ghcr.io/imgproxy/imgproxy:latest`
- Configuration docs at https://docs.imgproxy.net/latest/configuration/options list env vars
- Key env vars: IMGPROXY_BIND (default :8080), IMGPROXY_KEY, IMGPROXY_SALT, IMGPROXY_QUALITY

## Compose
- Converted from docker run command in README
- Default port 8080, basic configuration with quality and resolution limits
