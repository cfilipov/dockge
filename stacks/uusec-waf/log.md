# UUSEC WAF

- **Category**: Web Servers / WAF
- **Source**: https://github.com/Safe3/uuWAF
- **Image**: `uusec/waf:latest`
- **Description**: Industry-leading free, high-performance AI and semantic Web Application Firewall (WAAP). Provides web attack detection, API security, and bot mitigation.
- **Ports**: Host network mode for WAF, 6612 for MySQL
- **Services**: uuwaf (WAF engine), wafdb (MySQL 5.7 backend)
- **Notes**: Derived from upstream docker/docker-compose.yml. Uses network_mode: host for the WAF service.
