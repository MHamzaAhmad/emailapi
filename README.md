# Email API Monorepo

A high-performance transactional email API built with a **Proto-First** architecture, using Go gRPC/Connect services and native gRPC SDKs.

## Quick Start

```bash
# Install dependencies
pnpm install
make install

# Start infrastructure (Postgres, ClickHouse, Redis)
make up

# Apply database migrations
make migrate

# Generate code from Protos (Single Source of Truth)
make proto

# Run the API (gRPC :9090 + HTTP :8080)
make dev-api
```

## Proto-First Architecture

All APIs, types, and SDKs are generated from Protocol Buffers defined in `proto/`. This ensures a single source of truth and type safety across all languages.

### Code Generation Pipeline

We use [Buf](https://buf.build) to manage protobuf generation. Running `make proto` triggers:

1.  **API Server (`apps/api/gen`)**:
    *   Go gRPC stubs
    *   gRPC-Gateway handlers (REST transcoding)
    *   OpenAPI v2 specification (`fern/openapi`)

2.  **SDKs (Connect)**:
    *   **TypeScript**: Connect v2 client + Protobuf-ES v2 types (`packages/sdk-ts`)
    *   **Go**: Connect-Go client + Protobuf types (`packages/sdk-go`)

3.  **Documentation**:
    *   Fern docs generated from the auto-generated OpenAPI spec (`make docs`)

## Structure

```
├── proto/             # Protocol Buffer definitions (Single Source of Truth)
│   ├── buf.yaml       # Buf module config
│   ├── buf.gen.yaml   # Generator config for API
│   └── buf.gen.sdk.yaml # Generator config for SDKs
├── apps/
│   ├── api/           # Go backend (gRPC + HTTP Gateway)
│   ├── web/           # Next.js dashboard
│   └── docs/          # Docusaurus documentation
├── packages/
│   ├── sdk-ts/        # TypeScript SDK (Connect v2)
│   └── sdk-go/        # Go SDK (Connect-Go)
├── db/
│   ├── migrations/    # PostgreSQL migrations
│   ├── queries/       # sqlc queries
│   └── clickhouse/    # ClickHouse schema
└── tools/             # Docker, sqlc, Atlas configs
```

## Available Commands

| Command | Description |
|---------|-------------|
| `make proto` | **Core Command**: Generates API, SDKs, and OpenAPI from protos |
| `make up` | Start Docker Compose services |
| `make down` | Stop Docker Compose services |
| `make gen` | Run all generators (proto + sqlc) |
| `make migrate` | Apply database migrations |
| `make test` | Run service layer tests |
| `make docs` | Regenerate API reference docs |
| `make docs-dev` | Preview documentation locally |

## SDK Usage

### TypeScript (Connect v2)

Native gRPC-web client using Connect v2.

```typescript
import { createClient } from '@email-api/sdk-ts'

const client = createClient({ apiKey: 'em_...' })

const response = await client.emails.sendEmail({
  from: 'sender@example.com',
  to: ['recipient@example.com'],
  subject: 'Hello',
  body: 'World',
})
```

### Go (Connect-Go)

Native gRPC usage with Connect.

```go
import (
    emailapi "github.com/emailapi/sdk-go"
    emailapiv1 "github.com/emailapi/sdk-go/gen/v1"
    "connectrpc.com/connect"
)

client := emailapi.NewClient("em_...")

resp, err := client.Emails.SendEmail(ctx, connect.NewRequest(&emailapiv1.SendEmailRequest{
    From:    "sender@example.com",
    To:      []string{"recipient@example.com"},
    Subject: "Hello",
    Body:    "World",
}))
```

## Services (Docker)

| Service | Port | Description |
|---------|------|-------------|
| gRPC Server | 9090 | Native gRPC API |
| HTTP Gateway | 8080 | REST API (gRPC-Gateway) |
| PostgreSQL | 5432 | Primary database |
| ClickHouse | 8123/9000 | Analytics database |
| Redis | 6379 | Caching & rate limiting |

## License

MIT
