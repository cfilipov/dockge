# Suroi

## Source
https://github.com/HasangerGames/suroi

## Description
Suroi is an open-source 2D battle royale web game inspired by surviv.io. Built with TypeScript, PixiJS, and Bun.

## Stack Details
- **server**: Game server using Bun runtime (oven/bun:1-alpine) on port 8000
- **client**: Web client serving the game UI on port 3000

## Configuration
- `SUROI_REGION`: Server region identifier (default NA)
- `SUROI_MAX_GAMES`: Maximum concurrent game instances

## Notes
- No official Docker image exists; using oven/bun base image as the project uses Bun
- Upstream project has no Dockerfile or docker-compose; this is a mock fixture layout
- The actual deployment requires building from source with `bun install` and `bun run build`
