# Samvera Hyrax Stack

## Source
- Repository: https://github.com/samvera/hyrax
- Compose file: https://github.com/samvera/hyrax/blob/main/docker-compose-dassie.yml

## Research
- docker-compose.yml includes docker-compose-dassie.yml (Dassie is Hyrax's test app)
- Original compose uses source bind mounts (.dassie/ directory, ./ for engine code)
- Removed source code bind mounts and build context; using pre-built ghcr.io/samvera/hyrax-dev image
- Removed Chrome/Selenium service (testing only)
- Removed env_file references (.dassie/.env) and inlined environment variables
- Changed Solr configset from custom hyraxconf to _default since custom configs require source checkout
- Added ALLOW_EMPTY_PASSWORD for Redis bitnami image

## Services
- **web**: Hyrax Rails application with Puma (port 3000)
- **worker**: Sidekiq background job processor
- **postgres**: PostgreSQL 15 database
- **fcrepo**: Fedora Commons 4.7.5 repository
- **fits**: File format identification service
- **memcached**: In-memory caching
- **redis**: Redis 6.2 for Sidekiq and caching
- **solr**: Solr 9.9 search engine

## Notes
- The ghcr.io/samvera/hyrax-dev image is a development image; production deployments build custom images
- Uses dedicated bridge network named br-hyrax
- Solr uses default configset; production should use Hyrax-specific Solr configs
- The web and worker services share volumes for derivatives, uploads, and Rails assets
