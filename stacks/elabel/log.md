# E-Label

## Sources
- Repository: https://github.com/filipecarneiro/ELabel
- Compose file: https://github.com/filipecarneiro/ELabel/blob/main/docker-compose.yml
- Docker image: fcarneiro/elabel (Docker Hub)

## Notes
- ASP.NET Core web application for EU wine e-label regulations
- Uses Microsoft SQL Server 2022 Express as the database
- Removed `version: '3.4'` field for Compose V2 compatibility
- Environment variables for admin password and DB password defined in .env file
- App listens on port 8080 internally, mapped to port 80
