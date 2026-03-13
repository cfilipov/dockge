# Unison

## Source
- GitHub: https://github.com/bcpierce00/unison
- Docker Hub: https://hub.docker.com/r/eugenmayer/unison

## Description
Unison is a file-synchronization tool for POSIX-compliant systems (e.g., BSDs, GNU/Linux, macOS) and Windows. It allows two replicas of a collection of files and directories to be stored on different hosts, modified separately, then brought up to date by propagating changes. The eugenmayer/unison Docker image provides a lightweight Unison server commonly used for bidirectional volume sync in development environments.

## Stack Components
- **unison**: Unison sync server (eugenmayer/unison:latest)

## Ports
- 5000: Unison sync protocol

## Volumes
- unison_data: Synchronized data directory

## Configuration Notes
- UNISON_DIR sets the sync target directory inside the container
- UNISON_OWNER/UNISON_GROUP set file ownership UID/GID
- Commonly used with docker-sync for macOS/Windows development environments
- The eugenmayer/unison image is the most popular Docker image for Unison file sync
