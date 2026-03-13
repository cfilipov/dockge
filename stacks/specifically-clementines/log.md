# Specifically Clementines

- **Source**: https://github.com/davideshay/groceries
- **Description**: Self-hosted grocery list app with real-time sync, multi-user support, and recipe integration
- **Architecture**: Node.js server with CouchDB for real-time sync and offline support
- **Images**: ghcr.io/davideshay/groceries-server (inferred), couchdb:3
- **Compose reference**: Constructed from README documentation and project architecture (no official docker-compose in repo)
- **Notes**: Formerly called "Groceries". Key feature is reliable real-time sync across devices using CouchDB. Supports list groups, per-store category sorting, and Tandoor recipe import. Has companion mobile apps.
