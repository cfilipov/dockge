# Sourcehut

- **Status**: skipped
- **Source**: https://sr.ht/
- **Description**: A network of open-source tools for software development (git hosting, mailing lists, CI, etc.)
- **Notes**: Sourcehut (sr.ht) is a modular suite of many microservices (git.sr.ht, builds.sr.ht, meta.sr.ht, etc.) that does not provide an official Docker image or docker-compose setup. The project officially recommends bare-metal or VM installation via Alpine packages. No reliable community Docker images exist that bundle the full suite. Individual service containers would require extensive custom configuration beyond what a simple compose file can provide.
