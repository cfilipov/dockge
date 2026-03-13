# SimpleLogin Stack

## Source
- GitHub: https://github.com/simple-login/app
- Image: simplelogin/app:4.6.2-beta

## What was done
- Converted docker run commands from README self-hosting guide to compose format
- 4 services: postgres, app (web), email-handler, job-runner
- Custom bridge network with fixed subnet matching docs
- Created pgdata/ and sl-data/ directories for bind mounts
- Added .env with required variables
