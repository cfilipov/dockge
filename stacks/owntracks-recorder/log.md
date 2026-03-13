# OwnTracks Recorder

- **Source**: https://github.com/owntracks/recorder
- **Description**: Lightweight program for storing and accessing OwnTracks location data
- **Compose reference**: Constructed from Docker Hub and README documentation
- **Services**: recorder (C backend with HTTP API)
- **Default port**: 8083 (HTTP API + web frontend)
- **Notes**: Connects to MQTT broker for receiving location updates; can also accept HTTP POST directly
