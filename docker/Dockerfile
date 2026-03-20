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
# Build Rust backend (embeds frontend)
############################################
FROM rust:1-bookworm AS build_backend
RUN rustup target add x86_64-unknown-linux-musl && \
    apt-get update && apt-get install -y musl-tools && rm -rf /var/lib/apt/lists/*
WORKDIR /app

# Copy frontend build output — rust-embed references ../dist/ from backend-rust/
COPY --from=build_frontend /app/dist ./dist/

# Copy Rust project files — dependencies first for caching
COPY backend-rust/Cargo.toml backend-rust/Cargo.lock ./backend-rust/
RUN mkdir -p backend-rust/src && echo 'fn main() {}' > backend-rust/src/main.rs && \
    cd backend-rust && cargo build --release --target x86_64-unknown-linux-musl && \
    rm -rf src

# Copy actual source and rebuild
COPY backend-rust/src ./backend-rust/src/
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/app/backend-rust/target \
    cd backend-rust && \
    touch src/main.rs && \
    cargo build --release --target x86_64-unknown-linux-musl && \
    cp target/x86_64-unknown-linux-musl/release/dockge-rust /dockge

############################################
# Final image — Alpine (bash + docker CLI)
############################################
FROM alpine:3 AS release

RUN apk add --no-cache bash

# Docker CLI + compose plugin for lifecycle commands and interactive exec.
COPY --from=docker:27-cli /usr/local/bin/docker /usr/local/bin/docker
COPY --from=docker:27-cli /usr/local/libexec/docker/cli-plugins/docker-compose /usr/local/libexec/docker/cli-plugins/docker-compose

WORKDIR /app
COPY --from=build_backend /dockge /app/dockge

VOLUME /app/data
EXPOSE 5001
HEALTHCHECK --interval=60s --timeout=30s --start-period=60s --retries=5 CMD wget -q -O /dev/null http://127.0.0.1:5001/healthz
ENTRYPOINT ["/app/dockge"]
