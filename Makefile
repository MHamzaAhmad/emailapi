.PHONY: up down gen proto migrate test docs clean

# Docker Compose commands
up:
	docker compose -f tools/docker-compose.yaml up -d

down:
	docker compose -f tools/docker-compose.yaml down

# Code generation (proto-first)
gen: proto gen-sqlc

proto:
	cd proto && buf dep update && buf generate
	cd proto && buf generate --template buf.gen.yaml
	cd proto && buf generate --template buf.gen.openapi.yaml	

gen-sqlc:
	cd tools && sqlc generate

# Database migrations (Goose)
# Postgres
migrate-pg-up:
	goose -dir db/postgres/migrations postgres "$(DATABASE_URL)" up

migrate-pg-down:
	goose -dir db/postgres/migrations postgres "$(DATABASE_URL)" down

migrate-pg-status:
	goose -dir db/postgres/migrations postgres "$(DATABASE_URL)" status

migrate-pg-create:
	@read -p "Migration name: " name; \
	goose -dir db/postgres/migrations postgres "$(DATABASE_URL)" create $$name sql

# ClickHouse
migrate-ch-up:
	goose -dir db/clickhouse/migrations clickhouse "$(CLICKHOUSE_URL)" up

migrate-ch-down:
	goose -dir db/clickhouse/migrations clickhouse "$(CLICKHOUSE_URL)" down

migrate-ch-status:
	goose -dir db/clickhouse/migrations clickhouse "$(CLICKHOUSE_URL)" status

migrate-ch-create:
	@read -p "Migration name: " name; \
	goose -dir db/clickhouse/migrations clickhouse "$(CLICKHOUSE_URL)" create $$name sql

# River
migrate-river:
	river migrate-up --database-url "$(DATABASE_URL)"
# Testing
test:
	cd apps/api && go test ./internal/service/... -v

test-all:
	cd apps/api && go test ./... -v

# Development
dev-api:
	cd apps/api && air

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
