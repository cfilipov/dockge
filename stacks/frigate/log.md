# Frigate

## Sources
- Official docs: https://docs.frigate.video/frigate/installation
- Docker image: ghcr.io/blakeblackshear/frigate:stable
- GitHub: https://github.com/blakeblackshear/frigate

## Notes
- Compose from official Frigate installation docs
- Device mappings commented out (hardware-specific: Coral USB/PCIe, Intel hwaccel, V4L2)
- Ports: 8971 (web UI), 8554 (RTSP), 8555 (WebRTC TCP+UDP)
- tmpfs cache for frame processing, shm_size 512mb
- Config directory bind-mounted at ./config (user must create config.yml)
- Storage for recordings/clips at ./storage
