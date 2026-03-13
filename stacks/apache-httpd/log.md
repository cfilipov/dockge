# Apache HTTP Server

- **Category**: Web Servers
- **Source**: https://hub.docker.com/_/httpd (Official Docker image)
- **Image**: `httpd:2.4`
- **Description**: The Apache HTTP Server is a free, open-source, cross-platform web server. One of the most popular web servers in the world, playing a key role in the growth of the World Wide Web.
- **Ports**: 8080 -> 80 (HTTP)
- **Volumes**: `./public-html` for web content, `./httpd.conf` for config
- **Config files**: `httpd.conf` (bind-mounted)
