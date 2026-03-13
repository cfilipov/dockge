# miniserve

## Source
- GitHub: https://github.com/svenstaro/miniserve
- Docker Hub: https://hub.docker.com/r/svenstaro/miniserve

## Description
For when you really just want to serve some files over HTTP right now! A small, self-contained cross-platform CLI tool for serving files and directories over HTTP. Written in Rust.

## Stack
- svenstaro/miniserve:latest — main application (port 8080)

## Notes
- Configured via command-line arguments in the `command` directive
- --upload-files enables file upload support
- --mkdir enables directory creation
- Supports authentication via --auth flag (username:password)
- No persistent state — purely serves files from the mounted volume
