# Documenso

## What was done
- Based on official production compose from documenso/documenso release branch
- Services: documenso (Next.js app), PostgreSQL
- Images: documenso/documenso:latest, postgres:15
- Port: 3000 (web UI)
- Signing certificate mount at /opt/documenso/cert.p12
- All secrets extracted to .env
