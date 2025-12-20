---
sidebar_position: 1
---

# TypeScript SDK

Native gRPC client using Connect v2 for browser and Node.js.

## Installation

```bash
npm install @email-api/sdk-ts
# or
pnpm add @email-api/sdk-ts
```

## Setup

```typescript
import { createClient } from '@email-api/sdk-ts'

const client = createClient({
  apiKey: 'em_your_api_key',
  baseUrl: 'https://api.emailapi.dev', // optional
})
```

## Send Email

```typescript
const response = await client.emails.sendEmail({
  from: 'sender@example.com',
  to: ['recipient@example.com'],
  subject: 'Hello',
  body: 'Plain text content',
  html: '<h1>HTML content</h1>',
  metadata: { campaign: 'welcome' },
})

console.log(response.id) // Email ID
console.log(response.status) // 'pending', 'sent', etc.
```

## Get Email

```typescript
const email = await client.emails.getEmail({ id: 'email_123' })
```

## List Emails

```typescript
const { data, limit, offset } = await client.emails.listEmails({
  limit: 10,
  offset: 0,
})
```

## Webhooks

```typescript
// Create webhook
const webhook = await client.webhooks.createWebhook({
  name: 'My Webhook',
  url: 'https://example.com/webhook',
  events: ['EMAIL_EVENT_TYPE_DELIVERED', 'EMAIL_EVENT_TYPE_BOUNCED'],
})

// List webhooks
const { data } = await client.webhooks.listWebhooks({})
```
