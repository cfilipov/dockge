############################################
# Build frontend
############################################
FROM node:22-bookworm-slim AS build_frontend
WORKDIR /app

# Install dependencies first for caching
COPY package.json pnpm-lock.yaml .npmrc ./
RUN corepack enable && pnpm install --frozen-lockfile

# Copy frontend source and shared common/ types
COPY frontend/ ./frontend/
COPY common/ ./common/
COPY tsconfig.json ./
RUN pnpm run build:frontend

# Strip pre-compressed variants — the Go server gzips on the fly,
# so .br and .gz files only add ~800KB to the embedded binary.
RUN find frontend-dist -name '*.br' -o -name '*.gz' | xargs rm -f

############################################
# Build Go backend (embeds frontend)
############################################
FROM golang:1.25-alpine AS build_backend
RUN apk add --no-cache git upx
WORKDIR /app

# Copy go module files first for dependency caching
COPY backend-go/go.mod backend-go/go.sum ./backend-go/
RUN cd backend-go && go mod download

# Copy frontend build output for embedding (pre-compressed files already stripped)
COPY --from=build_frontend /app/frontend-dist ./backend-go/frontend-dist/

# Copy package.json for version extraction, then Go source
COPY package.json ./
COPY backend-go/ ./backend-go/
RUN VERSION=$(grep '"version"' package.json | head -1 | sed 's/.*"\([0-9][^"]*\)".*/\1/') && \
    cd backend-go && CGO_ENABLED=0 go build \
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
