# GoSE

- **Source**: https://codeberg.org/stv0g/gose
- **Category**: File Transfer - Single-click & Drag-n-drop Upload
- **Description**: Modern file uploader focusing on scalability and simplicity. Uses S3 backend for storage with content-hash deduplication and multi-part uploads.
- **Image(s)**: `ghcr.io/stv0g/gose:v0.4.0`, `minio/minio:latest`
- **Compose source**: Adapted from upstream `compose.yaml`
- **Notes**: Requires S3-compatible storage (MinIO included). Supports AWS S3, Ceph RadosGW, and MinIO. Up to 50GB uploads with 16MB parts.
