# ANALOG
**Project:** https://github.com/orangecoloured/analog
**Source:** https://github.com/orangecoloured/analog
**Status:** partial
**Compose source:** Converted from docker run command in README

## What was done
- Created compose.yaml based on the docker run example in the README
- Added Redis as the database backend (one of the supported options)
- No pre-built Docker image exists on Docker Hub or ghcr.io; used node:20-alpine as base

## Issues
- No official Docker image published; the project only provides a Dockerfile for self-building
- Used node:20-alpine as a placeholder since no real registry image exists for this project
