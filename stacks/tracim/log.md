# Tracim

## Overview
Tracim is a collaborative platform for team collaboration. It provides document management, knowledge base, threaded discussions, and task tracking. Designed for both technical and non-technical teams.

## Image
- **Docker Hub**: `algoo/tracim`
- **Source**: https://github.com/tracim/tracim
- **Pulls**: ~35K
- **Tags**: latest (actively maintained)

## Stack Details
- Tracim application server on port 80
- PostgreSQL for database (also supports SQLite for testing)
- Persistent volumes for configuration and user data

## Notes
- Default login: admin@admin.admin / admin@admin.admin
- Supports SQLite (for testing) or PostgreSQL (for production)
- MIT licensed
- Email notifications can be enabled via environment variables
- Base URL must be set correctly for links in notifications
