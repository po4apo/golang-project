SHELL := bash

# Database configuration
AUTH_DB_DSN ?= postgres://authuser:authpass@localhost:5432/authdb?sslmode=disable

# Deployment configuration
SERVER_HOST ?= 88.218.169.245
SERVER_USER ?= root
DEPLOY_PATH ?= /opt/golang-project

.PHONY: help generate lint test test-coverage build dev migrate-auth-up migrate-auth-down migrate-auth-status run-auth run-rest build-auth build-rest docker-up docker-down docker-build docker-logs docker-restart docker-clean docker-dev-up docker-dev-down docker-prod-up docker-prod-down deploy-manual deploy-check server-setup ci-lint ci-test

help:
	@echo "Available commands:"
	@echo "  make generate          - Generate protobuf files"
	@echo "  make lint              - Run linter"
	@echo "  make test              - Run tests"
	@echo "  make test-coverage     - Run tests with coverage"
	@echo "  make build             - Build all services"
	@echo "  make docker-dev-up     - Start development environment"
	@echo "  make docker-prod-up    - Start production environment"
	@echo "  make deploy-manual     - Manual deploy to server"
	@echo "  make server-setup      - Setup server for deployment"

generate:
	cd api/proto && buf generate

lint:
	golangci-lint run ./... --timeout=5m

ci-lint:
	@echo "Running CI lint checks..."
	@if [ -n "$$(gofmt -s -l .)" ]; then \
		echo "Code is not formatted:"; \
		gofmt -s -d .; \
		exit 1; \
	fi
	golangci-lint run ./... --timeout=5m

test:
	go test -v ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

ci-test:
	@echo "Running CI tests..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

build:
	go build ./...

dev:
	@echo "Starting development environment..."

migrate-auth-up:
	@echo "Applying auth migrations..."
	migrate -database "$(AUTH_DB_DSN)" -path services/auth-service/migrations up

migrate-auth-down:
	@echo "Rolling back auth migrations..."
	migrate -database "$(AUTH_DB_DSN)" -path services/auth-service/migrations down 1

migrate-auth-status:
	@echo "Checking auth migration status..."
	migrate -database "$(AUTH_DB_DSN)" -path services/auth-service/migrations version

build-auth:
	@echo "Building auth-service..."
	go build -o services/auth-service/bin/auth-service ./services/auth-service/cmd/main.go

build-rest:
	@echo "Building rest-api..."
	go build -o services/rest-api/bin/rest-api ./services/rest-api/cmd/main.go

run-auth:
	@echo "Starting auth-service..."
	./services/auth-service/bin/auth-service

run-rest:
	@echo "Starting rest-api..."
	./services/rest-api/bin/rest-api

# Docker команды
docker-build:
	@echo "Building Docker images..."
	cd deploy/docker-compose && docker-compose build

docker-up:
	@echo "Starting all services with Docker Compose..."
	cd deploy/docker-compose && docker-compose up -d
	@echo "Services started! REST API available at http://localhost:8080"
	@echo "gRPC available at localhost:50051"

docker-down:
	@echo "Stopping all services..."
	cd deploy/docker-compose && docker-compose down

docker-logs:
	@echo "Showing logs..."
	cd deploy/docker-compose && docker-compose logs -f

docker-restart:
	@echo "Restarting all services..."
	cd deploy/docker-compose && docker-compose restart

docker-clean:
	@echo "Removing all containers, volumes and images..."
	cd deploy/docker-compose && docker-compose down -v --rmi all

# Development environment
docker-dev-up:
	@echo "Starting development environment..."
	cd deploy/docker-compose && docker-compose -f docker-compose.dev.yml up -d
	@echo "Development environment started!"
	@echo "REST API: http://localhost:8080"
	@echo "gRPC: localhost:50051"
	@echo "PostgreSQL: localhost:5432"

docker-dev-down:
	@echo "Stopping development environment..."
	cd deploy/docker-compose && docker-compose -f docker-compose.dev.yml down

docker-dev-logs:
	cd deploy/docker-compose && docker-compose -f docker-compose.dev.yml logs -f

# Production environment (local)
docker-prod-up:
	@echo "Starting production environment..."
	cd deploy/docker-compose && docker-compose -f docker-compose.production.yml up -d
	@echo "Production environment started!"

docker-prod-down:
	@echo "Stopping production environment..."
	cd deploy/docker-compose && docker-compose -f docker-compose.production.yml down

docker-prod-logs:
	cd deploy/docker-compose && docker-compose -f docker-compose.production.yml logs -f

# Deployment commands
deploy-manual:
	@echo "Starting manual deployment to $(SERVER_HOST)..."
	./scripts/manual-deploy.sh -h $(SERVER_HOST) -u $(SERVER_USER) -p $(DEPLOY_PATH)

deploy-check:
	@echo "Checking deployment status..."
	@curl -f http://$(SERVER_HOST):8080/health || echo "Service is not responding"
	@ssh $(SERVER_USER)@$(SERVER_HOST) "cd $(DEPLOY_PATH) && docker ps"

server-setup:
	@echo "Setting up server at $(SERVER_HOST)..."
	scp scripts/server-setup.sh $(SERVER_USER)@$(SERVER_HOST):/tmp/
	ssh $(SERVER_USER)@$(SERVER_HOST) "cd /tmp && chmod +x server-setup.sh && sudo ./server-setup.sh"

# Helper commands
clean:
	@echo "Cleaning build artifacts..."
	rm -rf services/*/bin
	rm -f coverage.out coverage.html
	go clean ./...

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

fmt:
	@echo "Formatting code..."
	gofmt -s -w .
	go mod tidy
