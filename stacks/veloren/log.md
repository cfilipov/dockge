# Veloren

## Source
https://gitlab.com/veloren/veloren (mirrored at https://github.com/veloren/veloren)

## Description
Veloren is an open-source multiplayer voxel RPG inspired by games like Cube World, Zelda: Breath of the Wild, Dwarf Fortress, and Minecraft. This stack runs the dedicated game server.

## Stack Details
- **game-server**: Veloren server CLI (registry.gitlab.com/veloren/veloren/server-cli) on ports 14004-14006

## Ports
- 14004/tcp: Main game port
- 14005/tcp: Auth server port
- 14006/udp: Query port

## Configuration
- `userdata/`: Bind-mounted directory for world saves, settings, and server configuration
- `RUST_LOG`: Rust logging level configuration

## Notes
- Based on official docker-compose.yml from the Veloren repository
- Image hosted on GitLab Container Registry (not Docker Hub)
- `weekly` tag provides automated weekly builds
- Upstream compose includes Watchtower for auto-updates (omitted here for simplicity)
