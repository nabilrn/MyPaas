.PHONY: dev test lint build migrate-up sqlc help clean

# Variables
BACKEND_DIR := backend
FRONTEND_DIR := frontend
DOCKER_COMPOSE := docker-compose -f docker-compose.dev.yml

help:
	@echo "MyPaas — Available targets:"
	@echo ""
	@echo "  dev              Start development environment (dependencies + auto-reload)"
	@echo "  test             Run all tests (backend + frontend)"
	@echo "  lint             Lint code (golangci-lint + frontend checks)"
	@echo "  build            Build backend binary + frontend"
	@echo "  migrate-up       Run database migrations up"
	@echo "  migrate-down     Roll back database migrations"
	@echo "  sqlc             Generate sqlc code from queries"
	@echo "  clean            Remove build artifacts and temporary files"
	@echo "  help             Show this help message"

# Development
dev: migrate-up
	@echo "Starting development environment..."
	$(DOCKER_COMPOSE) up -d
	@echo ""
	@echo "Backend services started. Run in separate terminals:"
	@echo "  make backend-dev    — Start Go API with live reload"
	@echo "  make frontend-dev   — Start SvelteKit dev server"

backend-dev:
	@cd $(BACKEND_DIR) && air

frontend-dev:
	@cd $(FRONTEND_DIR) && pnpm install && pnpm dev

# Testing
test: test-backend test-frontend

test-backend:
	@echo "Running backend tests..."
	@cd $(BACKEND_DIR) && go test -v -race ./...

test-frontend:
	@echo "Running frontend tests..."
	@cd $(FRONTEND_DIR) && pnpm test

test-coverage:
	@echo "Running tests with coverage..."
	@cd $(BACKEND_DIR) && go test -cover ./...

# Linting
lint: lint-backend lint-frontend

lint-backend:
	@echo "Linting backend..."
	@cd $(BACKEND_DIR) && golangci-lint run

lint-frontend:
	@echo "Checking frontend..."
	@cd $(FRONTEND_DIR) && pnpm check

# Building
build: build-backend build-frontend

build-backend:
	@echo "Building backend binary..."
	@cd $(BACKEND_DIR) && go build -o bin/mypaas-api cmd/api/main.go
	@echo "✓ Binary: $(BACKEND_DIR)/bin/mypaas-api"

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && pnpm install && pnpm build
	@echo "✓ Built: $(FRONTEND_DIR)/build"

# Database
migrate-up:
	@echo "Running migrations up..."
	@cd $(BACKEND_DIR) && migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	@echo "Rolling back migrations..."
	@cd $(BACKEND_DIR) && migrate -path migrations -database "$$DATABASE_URL" down

migrate-new:
	@read -p "Migration name: " name; \
	cd $(BACKEND_DIR) && migrate create -ext sql -dir migrations -seq $$name

# Code generation
sqlc:
	@echo "Generating sqlc code..."
	@cd $(BACKEND_DIR) && sqlc generate
	@echo "✓ Generated: $(BACKEND_DIR)/internal/db"

# Cleanup
clean:
	@echo "Cleaning up..."
	@rm -rf $(BACKEND_DIR)/bin
	@rm -rf $(BACKEND_DIR)/tmp
	@rm -rf $(FRONTEND_DIR)/build
	@rm -rf $(FRONTEND_DIR)/.svelte-kit
	@echo "✓ Cleaned"

# Infrastructure
docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-reset:
	$(DOCKER_COMPOSE) down -v
	@echo "✓ Database reset"
