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

############################################
# Build Go backend (embeds frontend)
############################################
FROM golang:1.24-alpine AS build_backend
RUN apk add --no-cache git
WORKDIR /app

# Copy go module files first for dependency caching
COPY backend-go/go.mod backend-go/go.sum ./backend-go/
RUN cd backend-go && go mod download

# Copy frontend build output for embedding
COPY --from=build_frontend /app/frontend-dist ./backend-go/frontend-dist/

# Copy Go source
COPY backend-go/ ./backend-go/
RUN cd backend-go && CGO_ENABLED=0 go build -ldflags="-s -w" -o /dockge-backend .

############################################
# Final image
############################################
FROM alpine:3.21 AS release

RUN apk add --no-cache \
    docker-cli \
    docker-cli-compose \
    ca-certificates

WORKDIR /app
COPY --from=build_backend /dockge-backend /app/dockge-backend
RUN mkdir ./data

VOLUME /app/data
EXPOSE 5001
HEALTHCHECK --interval=60s --timeout=30s --start-period=60s --retries=5 CMD wget -qO /dev/null http://127.0.0.1:5001/ || exit 1
CMD ["/app/dockge-backend"]
