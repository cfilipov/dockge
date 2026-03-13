# Jitsi Meet

## Source
- Repository: https://github.com/jitsi/docker-jitsi-meet
- docker-compose.yml: https://raw.githubusercontent.com/jitsi/docker-jitsi-meet/master/docker-compose.yml
- env.example: https://raw.githubusercontent.com/jitsi/docker-jitsi-meet/master/env.example

## Images
- jitsi/web (frontend)
- jitsi/prosody (XMPP server)
- jitsi/jicofo (focus component)
- jitsi/jvb (video bridge)

## Notes
- The original compose file has hundreds of environment variables; this is a simplified but functional version
- The original uses `${RESTART_POLICY:-unless-stopped}` and `${JITSI_IMAGE_VERSION:-unstable}`; we default to stable
- Config volumes default to `~/.jitsi-meet-cfg`
- Passwords should be generated using the project's `gen-passwords.sh` script
- Additional optional services (jibri, jigasi, etherpad, whiteboard) are available in the full compose
