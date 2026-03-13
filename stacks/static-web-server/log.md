# Static Web Server

- **Category**: Web Servers
- **Source**: https://github.com/static-web-server/static-web-server
- **Image**: `joseluisq/static-web-server:2`
- **Description**: Blazing fast, single-binary static file server written in Rust. Only ~4MB with built-in compression (gzip, brotli, zstd), cache-control, CORS, and directory listing support.
- **Ports**: 8787 -> 80 (HTTP)
- **Volumes**: `./public` for static files
