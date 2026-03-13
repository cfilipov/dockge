# SIP3 Stack

## Research
- GitHub org: sip3io (sip3-captain-ce, sip3-salto-ce, sip3-twig-ce)
- Official Docker images on Docker Hub: sip3/sip3-captain-ce, sip3/sip3-salto-ce, sip3/sip3-twig-ce, sip3/sip3-grafana
- Installation primarily via Ansible, but Docker images are available
- Architecture: Captain (capture) -> Salto (processing) -> Twig (UI) + MongoDB + Grafana

## Compose
- Multi-service setup with all SIP3 components
- Captain uses host networking for packet capture
- MongoDB 6 as the data store
- Twig UI on port 8080, Grafana on port 3000
- Minimal application.yml config files for each service
