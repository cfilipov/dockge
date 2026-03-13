# Mailcow Stack

## Source
- GitHub: https://github.com/mailcow/mailcow-dockerized
- Images: ghcr.io/mailcow/* (various)

## What was done
- Based on official docker-compose.yml from repo (18 services)
- Simplified volume mounts (named volumes instead of complex bind mounts)
- Kept all core services: unbound, mysql, redis, rspamd, clamd, php-fpm, sogo, dovecot, postfix, nginx, acme, watchdog, netfilter, dockerapi, olefy, ofelia, memcached
- Added .env with required database and hostname variables
- Used custom bridge network with fixed subnet for unbound DNS
