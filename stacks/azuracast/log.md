# AzuraCast

- **Source**: https://github.com/AzuraCast/AzuraCast
- **Docker image**: ghcr.io/azuracast/azuracast
- **Reference**: https://docs.azuracast.com/en/getting-started/installation/docker
- **Description**: Self-hosted web radio management suite. All-in-one image with built-in web server, database, Icecast/SHOUTcast, and Liquidsoap.
- **Ports**: 8080 (HTTP), 8443 (HTTPS), 8000-8050 (radio streams)
- **Volumes**: station data, MySQL, SHOUTcast, GeoIP, SFTPGo, backups (all named volumes)
