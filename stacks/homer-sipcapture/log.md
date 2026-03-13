# SIPCAPTURE Homer Stack

## Research
- GitHub: sipcapture/homer + sipcapture/homer7-docker
- Official Docker images from ghcr.io: sipcapture/homer-app, sipcapture/heplify-server
- Full compose examples in homer7-docker repo (heplify-server/hom7-prom-all/)
- HOMER is a SIP/VoIP capture and monitoring platform

## Compose
- Based on official homer7-docker compose examples (simplified)
- homer-app web UI on port 9080
- heplify-server for HEP capture on ports 9060/9061
- PostgreSQL 15 for data storage
- Grafana for dashboards on port 9030
- Removed monitoring extras (prometheus, alertmanager, loki, node-exporter) for simplicity
