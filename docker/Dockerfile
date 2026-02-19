############################################
# Healthcheck Binary
############################################
FROM cmcooper1980/dockge:build-healthcheck AS build_healthcheck

############################################
# Build Rust backend
############################################
FROM rust:1-alpine AS build_backend
RUN apk add --no-cache musl-dev build-base
WORKDIR /app

# Copy manifests first for dependency caching
COPY backend-rust/Cargo.toml backend-rust/Cargo.lock ./backend-rust/

# Create dummy main.rs to build and cache dependencies
RUN mkdir -p backend-rust/src \
    && echo "fn main() {}" > backend-rust/src/main.rs \
    && cd backend-rust && cargo build --release \
    && rm -rf src

# Copy real source and rebuild (only recompiles application code)
COPY backend-rust/src ./backend-rust/src
RUN touch backend-rust/src/main.rs \
    && cd backend-rust && cargo build --release

############################################
# Build frontend
############################################
FROM node:22-bookworm-slim AS build_frontend
WORKDIR /app

# Install dependencies first for caching
COPY package.json package-lock.json ./
RUN npm ci

# Copy frontend source and shared common/ types
COPY frontend/ ./frontend/
COPY common/ ./common/
COPY tsconfig.json ./
RUN npm run build:frontend

############################################
# ⭐ Main Image
############################################
FROM alpine:3.21 AS release

RUN apk add --no-cache \
    docker-cli \
    docker-cli-compose \
    dumb-init \
    ca-certificates

WORKDIR /app
COPY --from=build_healthcheck /app/extra/healthcheck /app/extra/healthcheck
COPY --from=build_backend /app/backend-rust/target/release/dockge-backend /app/dockge-backend
COPY --from=build_frontend /app/frontend-dist /app/frontend-dist
RUN mkdir ./data

VOLUME /app/data
EXPOSE 5001
HEALTHCHECK --interval=60s --timeout=30s --start-period=60s --retries=5 CMD extra/healthcheck
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/app/dockge-backend"]
