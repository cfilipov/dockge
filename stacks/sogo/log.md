# SOGo

## Overview
SOGo is a fully supported and trusted groupware server with a focus on scalability and open standards. It provides web-based email, calendar, contacts, and ActiveSync support.

## Image
- **Docker Hub**: `jenserat/sogo`
- **Source**: https://github.com/inverse-inc/sogo
- **Tags**: latest, nightly, activesync, activesync-nightly

## Stack Details
- SOGo application with Apache (bundled in image)
- MariaDB for user/calendar/contact data
- Memcached for session caching (bundled in jenserat image but external is cleaner)

## Notes
- Created by Inverse Inc., now maintained by Alinto
- CalDAV/CardDAV/ActiveSync support
- Integrates with LDAP, IMAP, SMTP
- The jenserat/sogo image bundles Apache and memcached
- For production, consider using with an IMAP server (Dovecot) and SMTP relay
