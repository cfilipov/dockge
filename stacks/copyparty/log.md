# copyparty

## Source
- GitHub: https://github.com/9001/copyparty
- Docker Hub: https://hub.docker.com/r/copyparty/ac

## Description
Portable file server with accelerated resumable uploads, deduplication, WebDAV, FTP, zeroconf, media indexer, video thumbnails, audio transcoding, and write-only folders.

## Stack
- copyparty/ac:latest — main application (port 3923)

## Notes
- The `ac` image is the recommended edition including components for thumbnails and audio transcoding
- Data stored in /w (uploads) and /cfg (configuration)
