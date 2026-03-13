# Appwrite — Research Log

## Sources checked
1. Official self-hosting docs at `https://appwrite.io/docs/advanced/self-hosting/installation`
2. Official compose file from `https://appwrite.io/install/compose`
3. Official .env file from `https://appwrite.io/install/env`

## Compose file origin
From the official Appwrite self-hosting installation page. The compose file is served at `https://appwrite.io/install/compose` and the .env at `https://appwrite.io/install/env`.

## Modifications
- Removed `version:` field (Compose V2 format)
- Trimmed some optional worker services (stats-resources, stats-usage, scheduler-executions, scheduler-messages, worker-migrations, browser) to reduce complexity while keeping the core functional stack
- Simplified environment variable lists to essential variables only
- Trimmed .env to essential variables (removed cloud storage provider keys, VCS integration, SMS settings)
- Removed HTTPS/TLS traefik labels (kept HTTP-only for simplicity)
