# Posio

## Source
https://github.com/abrenaut/posio

## Description
Posio is a multiplayer geography guessing game built with Python/Django and Redis. Players guess locations on a map and score points based on proximity.

## Stack Details
- **web**: Main Django web server serving the game UI on port 8000
- **gameloops**: Background process running game loop logic
- **redis**: Redis 7.2 for pub/sub messaging and game state

## Notes
- Upstream uses `build:` context; replaced with `python:3.12-slim-bookworm` base image for mock fixture
- SpatiaLite database stored in named volume
- Both web and gameloops services share the spatialite volume and connect to Redis
