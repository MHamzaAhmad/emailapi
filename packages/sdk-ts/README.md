# simpleemailapi

[![npm version](https://img.shields.io/npm/v/simpleemailapi.svg)](https://www.npmjs.com/package/simpleemailapi)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

Backend-first HTTP/2 gRPC client for Node.js 18+ and Bun. Type-safe email sending with real-time event streaming.

## Installation

```bash
npm install simpleemailapi
```

```bash
pnpm add simpleemailapi
```

```bash
bun add simpleemailapi
```

## Quick Start

```typescript
import { createClient } from 'simpleemailapi'

const client = createClient({ apiKey: 'em_...' })

// Send an email
const result = await client.send({
  from: 'hello@yourdomain.com',
  to: ['user@example.com'],
  subject: 'Hello!',
  body: 'World'
})

console.log(result.id, result.status)
```

## Real-time Event Streaming

Stream email events with typed callbacks - only handle the events you care about:

```typescript
const controller = client.onReceive({
  onDelivered: (event) => {
    console.log('Delivered to:', event.recipients)
  },
  onReplied: (event) => {
    console.log('Reply from:', event.from)
    console.log('Message:', event.body)
  },
  onBounced: (event) => {
    console.log('Bounced:', event.bounceType, event.recipients)
  },
  onError: (err) => {
    console.error('Stream error:', err)
  }
})

// Stop listening when done
controller.abort()
```

## Available Event Handlers

| Handler | Description |
|---------|-------------|
| `onSent` | Email accepted for delivery |
| `onDelivered` | Email delivered to recipient's mailbox |
| `onBounced` | Email bounced (hard or soft) |
| `onComplained` | Recipient marked email as spam |
| `onRejected` | Email rejected before sending |
| `onDelayed` | Email delivery delayed |
| `onReplied` | Reply received to a sent email |
| `onFailed` | Email sending failed permanently |
| `onError` | Stream error occurred |

## Service Clients

Access underlying service clients for advanced usage:

```typescript
// Email operations
await client.email.sendEmail({ ... })

// Domain management
await client.domains.listDomains({})
await client.domains.createDomain({ domain: 'example.com' })
```

## Requirements

- Node.js 18+ or Bun
- TypeScript 5.0+ (for type definitions)

## Documentation

Visit [simpleemailapi.dev](https://simpleemailapi.dev) for full API documentation.

## Support

For any issues, contact us at [support@simpleemailapi.dev](mailto:support@simpleemailapi.dev).

## License

MIT
