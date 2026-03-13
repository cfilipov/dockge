# Stirling-PDF

## What was done
- Based on official docker/compose/docker-compose.yml from Stirling-Tools/Stirling-PDF
- Replaced build: directive with image: reference
- Single service - all-in-one PDF toolkit
- Image: docker.stirlingpdf.com/stirlingtools/stirling-pdf:latest
- Port: 8080
- Volumes for tessdata (OCR), configs, and logs
- Includes healthcheck from upstream
- No .env needed
