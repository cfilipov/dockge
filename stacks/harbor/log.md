# Harbor

## Sources
- Official docs: https://goharbor.io/docs/2.11.0/install-config/
- Installer script: https://github.com/goharbor/harbor/blob/main/make/install.sh
- Prepare script: https://github.com/goharbor/harbor/blob/main/make/prepare
- harbor.yml template: https://github.com/goharbor/harbor/blob/main/make/harbor.yml.tmpl
- Component images: https://github.com/goharbor/harbor/blob/main/make/photon/Makefile

## Notes
- Harbor is normally deployed by downloading the installer tarball and running `install.sh`,
  which uses the `goharbor/prepare` container to generate docker-compose.yml from harbor.yml config.
- This compose file represents the standard Harbor architecture based on documented components.
- Default login: admin / Harbor12345
- Images all from goharbor org on Docker Hub, version v2.11.2.
- Services: log, registry, registryctl, postgresql, core, portal, jobservice, redis, nginx proxy.
- The official deployment method requires running `prepare` first to generate config files in common/config/.
  This compose file references those bind mounts as the installer would create them.
