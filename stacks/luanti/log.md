# Luanti (formerly Minetest)

- **Source**: https://github.com/luanti-org/luanti
- **Description**: Open-source voxel game engine and game server
- **Image**: lscr.io/linuxserver/luanti:latest (LinuxServer.io, tags: 5.15.1)
- **Ports**: 30000/udp (game server)
- **Notes**: Luanti was renamed from Minetest. The LinuxServer.io image is the most maintained community image. Configuration is stored in the `/config/.luanti` volume. The `CLI_ARGS` environment variable passes command-line arguments to the server (e.g., `--gameid` to select which game to run).
