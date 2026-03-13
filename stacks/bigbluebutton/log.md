# BigBlueButton

- **Status**: skipped
- **Source**: https://github.com/bigbluebutton/bigbluebutton
- **Notes**: BigBlueButton is a complex multi-service video conferencing system that requires 15+ interconnected services (nginx, TURN, FreeSWITCH, Kurento, etc.), custom networking, and significant host configuration. The community Docker repo (bigbluebutton/docker) on the develop branch does not provide a standard docker-compose.yml. The official installation method uses a shell script (bbb-install.sh) on bare metal Ubuntu. Not suitable for a single compose stack.
