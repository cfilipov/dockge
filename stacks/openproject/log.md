# OpenProject

- **Status**: ok
- **Source**: https://github.com/opf/openproject
- **Image**: openproject/openproject:17
- **Notes**: Project management software. Uses the official all-in-one Docker image which bundles PostgreSQL, memcached, and the Rails app. The upstream dev docker-compose.yml uses `build:` contexts; the production deployment uses a separate repo (opf/openproject-deploy). This compose uses the simpler all-in-one image approach recommended in official docs.
