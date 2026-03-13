# Apache OpenMeetings

## Overview
Apache OpenMeetings is a web conferencing and collaboration platform. It provides video conferencing, instant messaging, whiteboard, document sharing, and screen sharing.

## Image
- **Docker Hub**: `apache/openmeetings`
- **Source**: https://github.com/apache/openmeetings
- **Tags**: 8.1.0, min-8.1.0, 8.0.0, 7.2.0, etc.

## Stack Details
- OpenMeetings application server on port 5443 (HTTPS)
- MariaDB for database storage
- Kurento Media Server for WebRTC video/audio processing
- STUN server configuration for NAT traversal

## Notes
- Apache Software Foundation project
- WebRTC-based video conferencing via Kurento
- Supports recording, screen sharing, whiteboard
- Default port is 5443 (HTTPS)
