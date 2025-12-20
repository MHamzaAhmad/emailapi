# Email API Monorepo

A high-performance transactional email API with Go backend, Next.js dashboard, and SDKs.

## Quick Start

```bash
# Install dependencies
pnpm install
make install

# Start infrastructure
make up

# Apply database migrations
make migrate

# Generate code
make gen

# Run the API
make dev-api
```

## Structure

```
├── apps/
│   ├── api/          # Go backend (Gin)
│   └── web/          # Next.js dashboard
├── packages/
│   ├── shared-spec/  # OpenAPI specification
│   ├── sdk-ts/       # TypeScript SDK
│   └── sdk-go/       # Go SDK
├── fern/             # SDK generation config
├── db/
│   ├── migrations/   # PostgreSQL migrations
│   ├── queries/      # sqlc queries
│   └── clickhouse/   # ClickHouse schema
└── tools/            # Docker, sqlc, Atlas configs
```

## Available Commands

| Command | Description |
|---------|-------------|
| `make up` | Start Docker Compose services |
| `make down` | Stop Docker Compose services |
| `make gen` | Generate code (sqlc, Fern) |
| `make migrate` | Apply database migrations |
| `make test` | Run service layer tests |
| `make docs` | Preview API documentation |

## Services (Docker)

| Service | Port | Description |
|---------|------|-------------|
| PostgreSQL | 5432 | Primary database |
| pgAdmin | 5050 | Database admin UI |
| ClickHouse | 8123/9000 | Analytics database |
| Tabix | 8124 | ClickHouse web UI |
| Redis | 6379 | Caching & rate limiting |

## SDK Usage

### TypeScript

```typescript
import { EmailApiClient } from '@email-api/sdk-ts'

const client = new EmailApiClient({ apiKey: 'em_...' })

const { id } = await client.emails.send({
  from: 'sender@example.com',
  to: ['recipient@example.com'],
  subject: 'Hello',
  body: 'World',
})
```

### Go

```go
import "github.com/emailapi/sdk-go"

client := emailapi.NewClient("em_...")

resp, _ := client.Emails.Send(ctx, &emailapi.SendEmailRequest{
    From:    "sender@example.com",
    To:      []string{"recipient@example.com"},
    Subject: "Hello",
    Body:    "World",
})
```

## License

MIT
