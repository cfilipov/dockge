# Outline Server

- **Status**: ok
- **Image**: quay.io/nicedream2/outline-shadowbox:latest
- **Note**: Outline Server (by Jigsaw/Google) is a Shadowsocks-based VPN proxy. The official image is hosted on quay.io. Typically deployed via the official install script (install_server.sh) which handles certificate generation and configuration. This compose file provides a manual setup — you may need to generate TLS certificates and configure shadowbox_config.json manually. Consider using the official install script for production: https://raw.githubusercontent.com/Jigsaw-Code/outline-server/master/src/server_manager/install_scripts/install_server.sh
- **Source**: https://github.com/Jigsaw-Code/outline-server
