############################################
# Build frontend
############################################
FROM node:22-bookworm-slim AS build_frontend
WORKDIR /app/web

# Install dependencies first for caching
COPY web/package.json web/pnpm-lock.yaml web/.npmrc ./
RUN corepack enable && pnpm install --frozen-lockfile

# Copy frontend source (includes src/common/ shared types)
COPY web/ ./
RUN pnpm run build

############################################
# Build Go backend (embeds frontend)
############################################
FROM golang:1.25-alpine AS build_backend
RUN apk add --no-cache git upx
WORKDIR /app

# Copy go module files first for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy frontend build output for embedding
COPY --from=build_frontend /app/dist ./dist/

# Copy package.json for version extraction, then Go source
COPY web/package.json ./web/
COPY internal/ ./internal/
COPY cmd/ ./cmd/
COPY main.go embed.go ./
RUN VERSION=$(grep '"version"' web/package.json | head -1 | sed 's/.*"\([0-9][^"]*\)".*/\1/') && \
    CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /dockge . && \
    upx --best --lzma /dockge

############################################
# Compress docker CLI + compose plugin with UPX
############################################
FROM golang:1.25-alpine AS compress_docker
RUN apk add --no-cache upx
COPY --from=docker:27-cli /usr/local/bin/docker /docker
COPY --from=docker:27-cli /usr/local/libexec/docker/cli-plugins/docker-compose /docker-compose
RUN upx --best --lzma /docker && upx --best --lzma /docker-compose

############################################
# Final image — Alpine (bash + docker CLI)
############################################
FROM alpine:3 AS release

RUN apk add --no-cache bash

# Docker CLI + compose plugin for lifecycle commands and interactive exec.
# UPX-compressed to keep image small.
COPY --from=compress_docker /docker /usr/local/bin/docker
COPY --from=compress_docker /docker-compose /usr/local/libexec/docker/cli-plugins/docker-compose

WORKDIR /app
COPY --from=build_backend /dockge /app/dockge

VOLUME /app/data
EXPOSE 5001
HEALTHCHECK --interval=60s --timeout=30s --start-period=60s --retries=5 CMD ["/app/dockge", "healthcheck"]
ENTRYPOINT ["/app/dockge"]
