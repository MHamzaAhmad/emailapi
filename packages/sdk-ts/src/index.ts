// Email API TypeScript SDK
// Native gRPC client using Connect v2

export { createClient, type EmailApiClient, type EmailApiClientConfig } from './client'

// Re-export generated message types
export * from './generated/v1/email_pb'
export * from './generated/v1/user_pb'
export * from './generated/v1/webhook_pb'
