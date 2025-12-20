.PHONY: up down gen proto migrate test docs clean

# Docker Compose commands
up:
	docker compose -f tools/docker-compose.yaml up -d

dev: 
	docker compose -f tools/docker-compose.yaml up --build

down:
	docker compose -f tools/docker-compose.yaml down

# Code generation (proto-first)
gen: proto gen-sqlc

proto:
	cd proto && buf dep update && buf generate
	cd proto && buf generate --template buf.gen.sdk.yaml

gen-sqlc:
	cd tools && sqlc generate

# Database migrations (declarative schema workflow)
migrate:
	atlas migrate apply --env local --config "file://tools/atlas.hcl"

migrate-diff:
	@read -p "Migration name: " name; \
	atlas migrate diff $$name --env local --config "file://tools/atlas.hcl"

migrate-status:
	atlas migrate status --env local --config "file://tools/atlas.hcl"

migrate-hash:
	atlas migrate hash --config "file://tools/atlas.hcl"

migrate-down:
	atlas migrate down --env local --config "file://tools/atlas.hcl"

# Testing
test:
	cd apps/api && go test ./internal/service/... -v

test-all:
	cd apps/api && go test ./... -v

# Development
dev-api:
	cd apps/api && go run main.go

dev-web:
	cd apps/web && pnpm dev

# Lint protos
lint-proto:
	cd proto && buf lint

# Docusaurus docs
docs:
	cd apps/docs && pnpm docusaurus gen-api-docs all

docs-dev:
	cd apps/docs && pnpm start

docs-build:
	cd apps/docs && pnpm build

# Clean generated files
clean:
	rm -rf apps/api/gen
	rm -rf packages/sdk-ts/src/generated
	rm -rf packages/sdk-go/generated
	rm -rf db/gen
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
lint: lint-proto
	cd apps/api && go vet ./...
	pnpm lint
