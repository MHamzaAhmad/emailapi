/**
 * Email API TypeScript SDK
 * 
 * Backend-first HTTP/2 gRPC client for Node.js 18+ and Bun.
 * 
 * All types are generated from Protocol Buffers (source of truth).
 * 
 * @example
 * ```ts
 * import { 
 *   createClient,
 *   EventType,           // enum
 *   EmailStatus,         // enum
 *   type SendEmailRequest,
 *   type SendEmailResponse,
 *   type Event,
 *   type Domain,
 * } from '@emailapi/sdk'
 * 
 * const client = createClient({ apiKey: 'em_...' })
 * 
 * await client.send({
 *   from: 'hello@yourdomain.com',
 *   to: ['user@example.com'],
 *   subject: 'Hello!',
 *   body: 'World'
 * })
 * ```
 * 
 * @packageDocumentation
 */

// Main client factory and all types
export {
    createClient,
    type ClientOptions,
    type EmailApiClient,
} from './client'

// Re-export all proto types from client
// This allows users to import everything from the package root:
// import { createClient, EventType, SendEmailRequest } from '@emailapi/sdk'
export * from './client'
