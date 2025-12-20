.PHONY: up down gen migrate test docs clean

# Docker Compose commands
up:
	docker compose -f tools/docker-compose.yaml up -d

down:
	docker compose -f tools/docker-compose.yaml down

# Code generation
gen: gen-sqlc gen-fern

gen-sqlc:
	cd tools && sqlc generate

gen-fern:
	fern generate

# Database migrations
migrate:
	atlas migrate apply --env local

migrate-new:
	@read -p "Migration name: " name; \
	atlas migrate new $$name --env local

# Testing
test:
	cd apps/api && go test ./internal/service/... -v

test-all:
	cd apps/api && go test ./... -v

# Documentation
docs:
	fern docs dev

# Development
dev-api:
	cd apps/api && go run main.go

dev-web:
	cd apps/web && pnpm dev

# Clean generated files
clean:
	rm -rf db/gen
	rm -rf packages/sdk-ts/src/generated
	rm -rf packages/sdk-go/generated
	rm -rf .nx

# Install dependencies
install:
	pnpm install
	cd apps/api && go mod download
	cd packages/sdk-go && go mod download

# Format code
fmt:
	cd apps/api && go fmt ./...
	pnpm exec prettier --write .

# Lint
lint:
	cd apps/api && go vet ./...
	pnpm lint
