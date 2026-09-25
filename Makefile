.PHONY: all help dev dev-ui build build-web build-bin test test-go test-web infra-up infra-down infra-logs clean

# Default target
all: help

help: ## Show this help message
	@echo ""
	@echo "  ⚡ Cubit — Developer Command Center"
	@echo "  =================================="
	@echo "  Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
	@echo ""

infra-up: ## Start background Docker infrastructure (Garage S3, Traefik, celld)
	docker compose -f deploy/docker-compose.yml up -d garage traefik celld

infra-down: ## Stop all Docker infrastructure containers
	docker compose -f deploy/docker-compose.yml down

infra-logs: ## Follow Docker infrastructure logs
	docker compose -f deploy/docker-compose.yml logs -f

build-web: ## Install npm packages and build React 19 web assets into web/dist
	cd web && npm install && npm run build

build-bin: ## Compile cubitd Go binary into bin/cubitd
	mkdir -p bin
	go build -o bin/cubitd cmd/cubitd/main.go

build: build-web build-bin ## Build both web frontend and cubitd Go binary

dev: infra-up ## Single command: start infra, build web assets (if needed), and run cubitd at :8000
	@if [ ! -d "web/dist" ]; then \
		echo "==> Building web frontend..."; \
		$(MAKE) build-web; \
	fi
	@echo "==> Starting cubitd control plane at http://localhost:8000..."
	go run cmd/cubitd/main.go

dev-ui: ## Run Vite dev server with Hot Module Replacement (HMR) at :5173
	cd web && npm run dev

test-go: ## Run all Go unit and integration tests with race detector
	go test -race -v ./...

test-web: ## Run frontend Vitest test suites
	cd web && npm test

test: test-go test-web ## Run all backend and frontend tests

clean: ## Clean build artifacts, binaries, and local SQLite database
	rm -rf bin/
	rm -rf web/dist/
	rm -f cubit.db cubit.db-shm cubit.db-wal
