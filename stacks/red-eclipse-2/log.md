# Red Eclipse 2

## Source
https://github.com/redeclipse/base

## Description
Red Eclipse 2 is a free, open-source FPS game built on the Tesseract engine. This stack runs a dedicated game server.

## Stack Details
- **server**: Red Eclipse dedicated server (iceflower/redeclipse-server) on port 28801 (TCP+UDP)

## Configuration
- `servinit.cfg`: Server configuration (name, max players, admin settings)
- Environment variables for server name, description, and passwords

## Notes
- Official repo has no Docker image; using community image `iceflower/redeclipse-server`
- Port 28801 used for both TCP and UDP game traffic
- Server data persisted in named volume
