# Koha ILS - Research Log

## Sources
1. **Project homepage**: https://koha-community.org/demo/ - No Docker instructions on demo page
2. **Docker Hub (official testing)**: https://hub.docker.com/r/koha/koha-testing - 406k+ pulls, official community image
3. **Docker Hub (elasticsearch-icu)**: https://hub.docker.com/r/koha/elasticsearch-icu - 240k+ pulls, companion image
4. **GitLab (koha-testing-docker)**: https://gitlab.com/koha-community/koha-testing-docker/-/blob/main/docker-compose.yml - Official compose file
5. **Docker Hub (pinokew/koha)**: Production-oriented alternative (764 pulls)
6. **GitHub (pinokew/koha-deploy)**: https://github.com/pinokew/koha-deploy - Production compose with Cloudflare tunnel

## Notes
- The official `koha/koha-testing` image is primarily for development/testing but is the most widely used
- The official compose requires `SYNC_REPO` to mount local Koha source code; adapted for standalone use
- Services: MariaDB (database), Elasticsearch with ICU (search), Memcached (caching), Koha (application)
- The original compose also includes Selenium for test automation; removed for general use
- Staff interface (intranet) on port 8081, public catalog (OPAC) on port 8080
- Production alternative exists at pinokew/koha-deploy with MariaDB 11, RabbitMQ, Cloudflare tunnel
- Koha community primarily uses GitLab, not GitHub
