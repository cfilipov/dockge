# Chartbrew Stack

- Based on official docker-compose.yml from chartbrew/chartbrew repo
- Replaced `build: .` with `chartbrew/chartbrew:latest` image
- MySQL 8.4, Redis Alpine, and Chartbrew app
- Ports 4018 (API) and 4019 (frontend)
- Health check on MySQL before Chartbrew starts
