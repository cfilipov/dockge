# SeaweedFS

## Sources
- Official compose file: https://github.com/seaweedfs/seaweedfs/blob/master/docker/seaweedfs-compose.yml
- Repository: https://github.com/seaweedfs/seaweedfs

## Notes
- Compose taken directly from official repo `docker/seaweedfs-compose.yml`
- Removed `version:` field and prometheus sidecar (optional monitoring, not core)
- Image: `chrislusf/seaweedfs`
- Services: master (9333), volume (8080), filer (8888), s3 (8333), webdav (7333)
- Each service also exposes metrics ports in the 932x range
