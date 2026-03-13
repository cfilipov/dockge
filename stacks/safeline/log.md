# SafeLine

- **Category**: Web Servers / WAF
- **Source**: https://github.com/chaitin/SafeLine
- **Image**: Various `safeline-*` images from Chaitin
- **Description**: AI-powered Web Application Firewall (WAF) with semantic analysis engine. Provides HTTP flood protection, bot mitigation, and web attack detection using deep learning.
- **Ports**: 9443 (Management UI)
- **Services**: postgres, mgt (management), detect (detector), tengine (NGINX fork)
- **Notes**: Multi-service stack with custom network and static IPs. Derived from upstream compose.yaml.
