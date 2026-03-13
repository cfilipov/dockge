# Easy!Appointments

## Sources
- GitHub: https://github.com/alextselegidis/easyappointments
- Repo compose (dev-only): https://raw.githubusercontent.com/alextselegidis/easyappointments/main/docker-compose.yml
- Community Docker image: https://hub.docker.com/r/jamrizzi/easyappointments

## Notes
- Official repo only has a dev-oriented docker-compose with build context (no pre-built image)
- No official Docker Hub image from the project maintainer
- Using community image `jamrizzi/easyappointments` as it's the most commonly referenced
- MySQL 8.0 matches the official repo's dev compose
- Database credentials adapted from the repo's compose file
- Added healthcheck to MySQL service
- This is a community-maintained Docker setup; official project does not publish Docker images
