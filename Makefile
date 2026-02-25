.PHONY: help setup build build-web build-go dev dev-web dev-go clean lint fmt test test-go test-e2e docker

BINARY := dockge

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

setup: ## Install all dependencies (first-time dev setup)
	cd web && pnpm install
	cd e2e && pnpm install
	cd e2e && npx playwright install chromium

build: build-web build-go ## Build everything

build-web: ## Build frontend (Vite → dist/)
	cd web && pnpm run build

build-go: ## Build Go backend
	go build -o $(BINARY) .

dev: build-go ## Run Go backend + Vite HMR (ports 5001 + 5000)
	trap 'kill 0' EXIT; \
	cd web && pnpm run dev & \
	./$(BINARY) --dev --mock --port 5001 --data-dir test-data --stacks-dir test-data/stacks & \
	wait

dev-web: ## Run Vite dev server only (port 5000)
	cd web && pnpm run dev

dev-go: build-go ## Run Go backend only in dev+mock mode (port 5001)
	./$(BINARY) --dev --mock --port 5001 --data-dir test-data --stacks-dir test-data/stacks

clean: ## Remove build artifacts
	rm -rf dist $(BINARY)

lint: ## Lint frontend and Go code
	cd web && pnpm run lint
	go vet ./...

fmt: ## Format frontend and Go code
	cd web && pnpm run fmt
	gofmt -w .

test: test-go test-e2e ## Run all tests

test-go: ## Run Go tests with race detector
	go test -race ./...

test-e2e: build-web build-go ## Run Playwright E2E tests
	cd e2e && pnpm test

docker: ## Build Docker image
	docker buildx build -f docker/Dockerfile -t cfilipov/dockge:latest --target release .
