# Stalwart Mail Server Stack

## Source
- GitHub: https://github.com/stalwartlabs/stalwart
- Docs: https://stalw.art/docs/install/platform/docker
- Image: stalwartlabs/stalwart:latest

## What was done
- Converted docker run command from official docs to compose format
- All-in-one image (SMTP, IMAP, POP3, JMAP, ManageSieve, web admin)
- Created data/ directory for persistent storage
- Default admin credentials shown in docker logs on first run
