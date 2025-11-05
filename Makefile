SHELL := bash

# Database configuration
AUTH_DB_DSN ?= postgres://authuser:authpass@localhost:5432/authdb?sslmode=disable

.PHONY: generate lint test build dev migrate-auth-up migrate-auth-down migrate-auth-status run-auth run-rest build-auth build-rest docker-up docker-down docker-build docker-logs docker-restart

generate:
	cd api/proto && buf generate

lint:
	golangci-lint run ./...

test:
	go test ./...

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
