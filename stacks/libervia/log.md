# Libervia

## Source
- Project: https://repos.goffi.org/libervia-web
- Docker compose source: https://repos.goffi.org/libervia-backend/raw-file/tip/docker/web-demo.yml
- Documentation: https://libervia.org

## Research
- The official web-demo.yml compose file was found in the libervia-backend repository under docker/web-demo.yml
- The compose setup includes 5 services: prosody (XMPP server), postgres (database), pubsub, backend, and web frontend
- Demo credentials: demo/demo
- Access at http://localhost:8880
- Images are development previews (not stable releases)
- Variable substitution added for configurability; defaults match the upstream demo values
