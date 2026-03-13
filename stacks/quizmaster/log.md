# Quizmaster

## Source
https://github.com/nymanjens/quizmaster

## Description
A web-app for conducting a quiz with a player answer submission page, a quiz display page, and a master control page. Supports many question types configured via YAML.

## Stack Details
- **app**: Quizmaster Play Framework application (nymanjens/quizmaster) on port 9000

## Configuration
- `quiz-config.yml`: Quiz questions, rounds, and answers
- `application.conf`: Play Framework settings (secret key, language, port)
- Master controls unlocked via `masterSecret` in quiz-config.yml

## Notes
- Based on official `docker-compose-prebuilt.yml` from the repository
- Config files bind-mounted read-only into the container
