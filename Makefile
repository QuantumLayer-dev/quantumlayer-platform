.PHONY: up down logs test clean dev migrate setup help

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## Show this help message
	@echo '${GREEN}QuantumLayer V2 - Development Commands${NC}'
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${NC} ${GREEN}<command>${NC}'
	@echo ''
	@echo 'Available commands:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  ${YELLOW}%-15s${NC} %s\n", $$1, $$2}'

setup: ## Initial setup - install dependencies and prepare environment
	@echo "${GREEN}Setting up QuantumLayer development environment...${NC}"
	@cp .env.example .env 2>/dev/null || echo "${YELLOW}.env already exists${NC}"
	@docker-compose pull
	@echo "${GREEN}Setup complete! Run 'make up' to start services${NC}"

up: ## Start all services in development mode
	@echo "${GREEN}Starting QuantumLayer services...${NC}"
	@docker-compose up -d
	@echo "${YELLOW}Waiting for services to be healthy...${NC}"
	@sleep 10
	@docker-compose ps
	@echo "${GREEN}Services are running! Check http://localhost:3000${NC}"

down: ## Stop all services
	@echo "${YELLOW}Stopping QuantumLayer services...${NC}"
	@docker-compose down
	@echo "${GREEN}Services stopped${NC}"

logs: ## Show logs from all services (follow mode)
	@docker-compose logs -f

logs-api: ## Show API service logs
	@docker-compose logs -f api

test: ## Run all tests
	@echo "${GREEN}Running tests...${NC}"
	@go test ./... -v
	@cd apps/web && npm test
	@echo "${GREEN}All tests passed!${NC}"

test-unit: ## Run unit tests only
	@go test ./... -short -v

test-integration: ## Run integration tests
	@go test ./... -run Integration -v

clean: ## Clean up everything (volumes, cache, etc.)
	@echo "${RED}Cleaning up all data and volumes...${NC}"
	@docker-compose down -v
	@rm -rf tmp/ logs/ .cache/
	@echo "${GREEN}Cleanup complete${NC}"

dev: ## Start services in development mode with hot reload
	@echo "${GREEN}Starting development mode with hot reload...${NC}"
	@docker-compose up

dev-api: ## Run API service with hot reload (requires Air)
	@cd apps/api && air -c .air.toml

dev-web: ## Run web frontend in dev mode
	@cd apps/web && npm run dev

migrate: ## Run database migrations
	@echo "${GREEN}Running database migrations...${NC}"
	@docker-compose exec -T postgres psql -U quantum -d quantumlayer -f /docker-entrypoint-initdb.d/001_init.sql
	@echo "${GREEN}Migrations complete${NC}"

migrate-down: ## Rollback last migration
	@echo "${YELLOW}Rolling back last migration...${NC}"
	@docker-compose exec postgres psql -U quantum -d quantumlayer -c "DROP SCHEMA quantum CASCADE;"
	@echo "${GREEN}Rollback complete${NC}"

db-shell: ## Open PostgreSQL shell
	@docker-compose exec postgres psql -U quantum -d quantumlayer

redis-cli: ## Open Redis CLI
	@docker-compose exec redis redis-cli

build: ## Build all services
	@echo "${GREEN}Building all services...${NC}"
	@docker-compose build
	@echo "${GREEN}Build complete${NC}"

lint: ## Run linters
	@echo "${GREEN}Running linters...${NC}"
	@golangci-lint run ./...
	@cd apps/web && npm run lint
	@echo "${GREEN}Linting complete${NC}"

fmt: ## Format all code
	@echo "${GREEN}Formatting code...${NC}"
	@go fmt ./...
	@cd apps/web && npm run format
	@echo "${GREEN}Formatting complete${NC}"

bench: ## Run benchmarks
	@go test -bench=. -benchmem ./...

coverage: ## Generate test coverage report
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "${GREEN}Coverage report generated: coverage.html${NC}"

docker-status: ## Check status of all containers
	@docker-compose ps

docker-logs-postgres: ## Show PostgreSQL logs
	@docker-compose logs -f postgres

docker-logs-redis: ## Show Redis logs
	@docker-compose logs -f redis

docker-logs-temporal: ## Show Temporal logs
	@docker-compose logs -f temporal

monitoring: ## Open monitoring dashboards
	@echo "${GREEN}Opening monitoring dashboards...${NC}"
	@echo "Grafana: http://localhost:3001 (admin/admin)"
	@echo "Prometheus: http://localhost:9090"
	@echo "Temporal UI: http://localhost:8080"
	@echo "MinIO Console: http://localhost:9001 (minioadmin/minioadmin)"
	@echo "NATS Monitor: http://localhost:8222"

seed: ## Seed database with sample data
	@echo "${GREEN}Seeding database with sample data...${NC}"
	@docker-compose exec postgres psql -U quantum -d quantumlayer -c "INSERT INTO organizations (name, slug) VALUES ('Demo Org', 'demo') ON CONFLICT DO NOTHING;"
	@echo "${GREEN}Seeding complete${NC}"

validate: ## Validate configuration files
	@echo "${GREEN}Validating configuration...${NC}"
	@docker-compose config
	@echo "${GREEN}Configuration is valid${NC}"

.DEFAULT_GOAL := help