---
sidebar_position: 2
---

# Go SDK

Native gRPC client using Connect-Go.

## Installation

```bash
go get github.com/emailapi/sdk-go
```

## Setup

```go
import (
    emailapi "github.com/emailapi/sdk-go"
)

client := emailapi.NewClient("em_your_api_key")

// Custom options
client := emailapi.NewClient("em_your_api_key",
    emailapi.WithBaseURL("https://api.emailapi.dev"),
)
```

## Send Email

```go
import (
    emailapiv1 "github.com/emailapi/sdk-go/gen/v1"
    "connectrpc.com/connect"
)

resp, err := client.Emails.SendEmail(ctx, connect.NewRequest(&emailapiv1.SendEmailRequest{
    From:    "sender@example.com",
    To:      []string{"recipient@example.com"},
    Subject: "Hello",
    Body:    "Plain text content",
    Html:    "<h1>HTML content</h1>",
    Metadata: map[string]string{
        "campaign": "welcome",
    },
}))
if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Msg.Id)     // Email ID
fmt.Println(resp.Msg.Status) // pending, sent, etc.
```

## Get Email

```go
resp, err := client.Emails.GetEmail(ctx, connect.NewRequest(&emailapiv1.GetEmailRequest{
    Id: "email_123",
}))
```

## List Emails

```go
resp, err := client.Emails.ListEmails(ctx, connect.NewRequest(&emailapiv1.ListEmailsRequest{
    Limit:  10,
    Offset: 0,
}))

for _, email := range resp.Msg.Data {
    fmt.Println(email.Id, email.Subject)
}
```

## Webhooks

```go
// Create webhook
resp, err := client.Webhooks.CreateWebhook(ctx, connect.NewRequest(&emailapiv1.CreateWebhookRequest{
    Name:   "My Webhook",
    Url:    "https://example.com/webhook",
    Events: []emailapiv1.WebhookEventType{
        emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_DELIVERED,
        emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_BOUNCED,
    },
}))
```
