# HomeGallery

- **Source**: https://github.com/xemle/home-gallery
- **Docker image**: xemle/home-gallery, xemle/home-gallery-api-server
- **Compose ref**: https://github.com/xemle/home-gallery/blob/master/docker-compose.yml
- **Description**: Self-hosted photo and video gallery with AI-powered similarity search and face detection
- **Services**: gallery (web UI + indexer), api (TensorFlow.js extraction server)
- **Notes**: API backend options: cpu, wasm, node (wasm is default for cross-platform). Photos mounted at /data/Pictures.
