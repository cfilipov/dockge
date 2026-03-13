# Mopidy

## Source
- GitHub: https://github.com/mopidy/mopidy
- Docker Image: wernight/mopidy

## Description
Mopidy is an extensible music server written in Python. It plays music from local disk, Spotify, SoundCloud, TuneIn, and more. Supports MPD clients and HTTP clients.

## Ports
- 6680: HTTP web interface
- 6600: MPD protocol

## Volumes
- music: Music library (read-only)
- data: Local data/database
- mopidy.conf: Configuration file

## Notes
- Popular extensions include Mopidy-Spotify, Mopidy-Local, Mopidy-MPD
- Web UI available via Mopidy-Iris or Mopidy-Muse extensions
