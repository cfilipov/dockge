# HyperSwitch

- **Source**: https://github.com/juspay/hyperswitch
- **Compose reference**: `docker-compose.yml` from official repo (512 lines, simplified)
- **Status**: ok
- **Services**: pg (postgres), redis-standalone, hyperswitch-server, hyperswitch-producer, hyperswitch-consumer, hyperswitch-control-center
- **Notes**: Simplified from the full 512-line compose. Removed migration_runner (requires build context with source code), mailhog (optional profile), and monitoring services. The server requires config files mounted at `./config/` - users need to clone the repo's config directory. Build directives replaced with registry images from docker.juspay.io.
