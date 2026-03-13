# Tine Groupware

## Overview
Tine is a modern open source groupware and CRM application. It provides email, calendar, contacts, tasks, file management, and project tracking with a rich web interface.

## Image
- **Docker Hub**: `tinegroupware/tine`
- **Source**: https://github.com/tine-groupware/tine
- **Pulls**: ~16K
- **Tags**: latest (actively maintained)

## Stack Details
- Tine application with PHP-FPM + Nginx (bundled)
- MariaDB for database storage
- Redis for caching and session storage
- Persistent volumes for user files and temp data

## Notes
- Based on Ubuntu 24.04
- Supports Redis for both caching and sessions (recommended over file-based)
- Setup user is separate from the login admin user
- Credential cache shared key is required for security
- Configurable via TINE20_* environment variables
