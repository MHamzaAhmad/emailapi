---
sidebar_position: 1
---

# Introduction

Email API is a high-performance transactional email API built with a **Proto-First** architecture.

## Features

- **Native gRPC**: High-performance binary protocol with HTTP/2
- **REST Gateway**: JSON API via gRPC-Gateway for backwards compatibility
- **Native SDKs**: Type-safe clients generated from Protocol Buffers
- **Webhooks**: Real-time event notifications
- **Analytics**: Built-in email tracking and analytics

## Quick Start

### 1. Get an API Key

Sign up at [emailapi.dev](https://emailapi.dev) and get your API key.

### 2. Install an SDK

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
  <TabItem value="ts" label="TypeScript" default>
    ```bash
    npm install @email-api/sdk-ts
    ```
  </TabItem>
  <TabItem value="go" label="Go">
    ```bash
    go get github.com/emailapi/sdk-go
    ```
  </TabItem>
</Tabs>

### 3. Send Your First Email

<Tabs>
  <TabItem value="ts" label="TypeScript" default>
    ```typescript
    import { createClient } from '@email-api/sdk-ts'

    const client = createClient({ apiKey: 'em_...' })

    const response = await client.emails.sendEmail({
      from: 'sender@example.com',
      to: ['recipient@example.com'],
      subject: 'Hello from Email API',
      body: 'Your first email!',
    })
    ```
  </TabItem>
  <TabItem value="go" label="Go">
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
        Subject: "Hello from Email API",
        Body:    "Your first email!",
    }))
    ```
  </TabItem>
</Tabs>
