# iodine

- **Status**: ok
- **Image**: adamant/iodine
- **Note**: Requires NET_ADMIN capability and /dev/net/tun device passthrough. Tunnels IP traffic over DNS requests. The image is Alpine-based (~6MB). Traffic is NOT encrypted — consider layering a VPN on top for security.
- **Source**: https://github.com/yarrick/iodine
- **Docker Hub**: https://hub.docker.com/r/adamant/iodine
