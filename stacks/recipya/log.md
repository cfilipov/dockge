# Recipya

- **Source**: https://github.com/reaper47/recipya
- **Description**: Clean, simple recipe manager focused on family use with web import, nutritional info, and measurement conversion
- **Architecture**: Go backend with embedded web UI, SQLite database
- **Images**: ghcr.io/reaper47/recipya
- **Compose reference**: Based on upstream deploy/Dockerfile and documentation (no official docker-compose provided)
- **Notes**: Single container, self-contained Go binary. Supports importing from Mealie, Tandoor, and Nextcloud Cookbook. Currently being rewritten in Rust. Cross-compiled for multiple platforms.
