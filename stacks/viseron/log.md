# Viseron

## Sources
- Official docs: https://viseron.netlify.app/docs/documentation/installation
- Docker image: roflcoopter/viseron:latest
- GitHub: https://github.com/roflcoopter/viseron

## Notes
- Compose from official Viseron installation documentation (standard 64-bit Linux variant)
- Added restart policy
- Port 8888 for web UI
- Multiple volume mounts for segments, snapshots, thumbnails, event clips, timelapse, config
- shm_size 1024mb for shared memory
- VAAPI, CUDA, Jetson Nano, and Raspberry Pi variants also available (different images/devices)
- Config is managed through built-in web editor at config.yaml
