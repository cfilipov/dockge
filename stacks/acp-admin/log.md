# ACP Admin

- **Status**: skipped
- **Source**: https://github.com/acp-admin/acp-admin
- **Notes**: Ruby on Rails web application for managing CSA/ACP/Solawi organizations. Has a Dockerfile in the repo but no published Docker image on Docker Hub or any registry. The Dockerfile uses Ruby 4.0.1 and builds from source. Would require `build:` directive which is not suitable for a pre-built compose fixture.
