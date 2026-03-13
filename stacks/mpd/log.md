# MPD (Music Player Daemon)

## Source
- GitHub: https://github.com/MusicPlayerDaemon/MPD
- Docker Image: vimagick/mpd

## Description
Music Player Daemon (MPD) is a flexible, powerful, server-side application for playing music. Through plugins and libraries it can play a variety of sound files while being controlled by its network protocol.

## Ports
- 6600: MPD protocol
- 8800: HTTP audio stream

## Volumes
- music: Music library (read-only)
- data: MPD database and state
- mpd.conf: Configuration file

## Notes
- Control via MPD clients (ncmpcpp, mpc, GMPC, etc.)
- HTTP streaming output configured on port 8800
