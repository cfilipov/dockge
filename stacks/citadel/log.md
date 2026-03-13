# Citadel

## Overview
Citadel is the premier open source platform for email, groupware, and content management. It provides email, calendaring, contacts, instant messaging, and collaboration in a single integrated package.

## Image
- **Docker Hub**: `citadeldotorg/citadel`
- **Source**: https://citadel.org
- **Pulls**: ~35K

## Stack Details
- Single container (all-in-one: Citadel server + WebCit web interface)
- Exposes HTTP (80), HTTPS (443), SMTP (25), IMAP (143)
- Persistent storage for Citadel data and runtime state

## Notes
- All-in-one container includes mail server, web interface, and groupware
- Default admin password should be changed immediately after first login
- Ports mapped to non-privileged defaults to avoid conflicts
