# Cozy Cloud

## Overview
Cozy Cloud is a personal cloud platform that lets you manage your data (files, contacts, calendars, emails) from a single place. The cozy-stack is the server-side component.

## Image
- **Docker Hub**: `cozy/cozy-stack`
- **Source**: https://github.com/cozy/cozy-stack
- **Tags**: latest, 1.6.44, 1.6.43, etc.

## Stack Details
- cozy-stack application server on port 8080 (API) and 6060 (admin)
- CouchDB for document storage
- Persistent volumes for data and CouchDB

## Notes
- CouchDB is the primary data store (NoSQL document database)
- Admin port 6060 provides management API
- Self-hosting documentation at https://docs.cozy.io
