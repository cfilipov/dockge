# Apache Guacamole

## Sources
- Official repo: https://github.com/apache/guacamole-server
- Official Docker docs: https://guacamole.apache.org/doc/gug/guacamole-docker.html
- Docker Hub images: guacamole/guacd, guacamole/guacamole

## Notes
- Guacamole is a clientless remote desktop gateway supporting VNC, RDP, SSH, and telnet
- Two official images: `guacamole/guacd` (backend daemon) and `guacamole/guacamole` (web frontend)
- Compose assembled from the official Docker documentation's docker run examples
- Requires a database (MySQL or PostgreSQL) for authentication and connection storage
- Database must be initialized with the Guacamole schema (run `docker run --rm guacamole/guacamole /opt/guacamole/bin/initdb.sh --mysql > initdb.sql` then import into MySQL)
- Web UI accessible at http://localhost:8080/guacamole/
- Default credentials: guacadmin / guacadmin
