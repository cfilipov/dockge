# Engity's Bifroest

## Sources
- Official repo: https://github.com/engity-com/bifroest
- Docker setup docs: https://bifroest.engity.org/setup/in-docker/
- Container image: https://github.com/engity-com/bifroest/pkgs/container/bifroest
- Systemd service file: https://raw.githubusercontent.com/engity-com/bifroest/v0.7.4/contrib/systemd/bifroest-in-docker.service

## Notes
- Bifroest is a highly customizable SSH server with OIDC and classic auth
- Official image: `ghcr.io/engity-com/bifroest:latest`
- Compose derived from the official systemd docker run command in contrib/systemd/bifroest-in-docker.service
- Requires Docker socket mount to manage user session containers
- Config directory at /etc/engity/bifroest holds configuration.yaml
- Data directory at /var/lib/engity/bifroest for persistent state
- SSH accessible on port 22
- Download sample config: `curl -sSLf https://raw.githubusercontent.com/engity-com/bifroest/v0.7.4/contrib/configurations/simple-inside-docker.yaml -o ./config/configuration.yaml`
