# Payload CMS

- **Source**: https://github.com/payloadcms/payload
- **Status**: skipped
- **Reason**: Payload CMS is an npm framework/library (`npm create payload-app`), not a standalone Docker service. No official Docker image exists on Docker Hub. Users scaffold their own Node.js app and bring their own Dockerfile. Cannot produce a generic compose.yaml without a custom application build.
