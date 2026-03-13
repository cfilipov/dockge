# Roundup Issue Tracker

## Source
- GitHub: https://github.com/roundup-tracker/roundup
- Compose: https://github.com/roundup-tracker/roundup/blob/master/scripts/Docker/docker-compose.yml

## Research
- Repo has a docker-compose.yml in scripts/Docker/ but it uses `build:` context (no published image)
- The Dockerfile builds from source (local_pip, local, or pypi)
- No official pre-built Docker Hub image found
- Status: **skipped** — compose requires building from source, no published Docker image
