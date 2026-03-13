# iRedMail Stack

## Source
- GitHub: https://github.com/iredmail/dockerized
- Image: iredmail/mariadb:stable

## What was done
- Converted docker run command from docs to compose format
- All-in-one image includes Postfix, Dovecot, Roundcube, MariaDB, etc.
- Created data/ and data-mysql/ directories for bind mounts
- Added .env with required configuration variables
