# Zimbra Collaboration

## Overview
Zimbra Collaboration is an enterprise email, calendar, and collaboration platform. It provides webmail, contacts, calendar, tasks, and document management.

## Image
- **Docker Hub**: `zimbra/zm-docker` (community/test image)
- **Source**: https://github.com/Zimbra

## Stack Details
- Single all-in-one container (Zimbra bundles its own LDAP, MTA, mailbox store)
- Exposes HTTP (80), HTTPS (443), SMTP (25), IMAP (143/993), Admin console (7071)
- Persistent volume for /opt/zimbra (all data and config)

## Notes
- Zimbra is primarily enterprise software; Docker support is for testing/development
- The all-in-one image is very large and resource-intensive
- Production deployments typically use the official installer on bare metal/VMs
- Admin console accessible on port 7071
- Hostname must be a valid FQDN for mail to work properly
- This is a simplified test fixture; production Zimbra is significantly more complex
