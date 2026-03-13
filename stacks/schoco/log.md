# Schoco

- **Source**: https://github.com/PhiTux/schoco
- **Images**: phitux/schoco-backend, phitux/schoco-frontend (Docker Hub)
- **Description**: School Coding Helper - web-based Java IDE for teaching programming. Students write/compile/run Java in the browser with isolated Docker containers.
- **Services**: backend (Python API), frontend (Nginx reverse proxy), gitea (Git hosting for student code)
- **Compose reference**: Adapted from upstream docker-compose.yml
- **Notes**: Requires Docker socket access for spawning Java compilation containers. Gitea stores student repositories. Teacher key controls account creation.
