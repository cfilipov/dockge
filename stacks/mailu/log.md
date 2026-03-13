# Mailu Stack

## Source
- GitHub: https://github.com/Mailu/Mailu
- Docs: https://mailu.io
- Images: ghcr.io/mailu/* (v2.0)

## What was done
- Based on official test compose and documentation
- 9 services: redis, front (nginx), resolver (unbound), admin, imap (dovecot), smtp (postfix), antispam (rspamd), oletools, webmail (roundcube)
- Created mailu.env with core configuration
- Used named volumes for all persistent data
- Custom bridge network with fixed subnet for resolver
- Isolated noinet network for security-sensitive services
