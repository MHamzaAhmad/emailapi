/**
 * Email API TypeScript SDK
 * 
 * Backend-first HTTP/2 gRPC client for Node.js 18+ and Bun.
 * Uses Connect RPC for type-safe communication.
 * 
 * @example
 * ```ts
 * import { createClient, EventType } from '@emailapi/sdk'
 * 
 * const client = createClient({ apiKey: 'em_...' })
 * 
 * // Send an email
 * const result = await client.send({
 *   from: 'hello@yourdomain.com',
 *   to: ['user@example.com'],
 *   subject: 'Hello!',
 *   body: 'World'
 * })
 * 
 * console.log(result.id, result.status)
 * 
 * // Stream events in real-time
 * for await (const event of client.onReceive({ cursor: '0' })) {
 *   if (event.type === EventType.EMAIL_DELIVERED) {
 *     console.log('Delivered to:', event.payload.value?.recipients)
 *   }
 * }
 * ```
 * 
 * @packageDocumentation
 */

import { createClient as createConnectClient, type Client } from '@connectrpc/connect'
import { createGrpcTransport } from '@connectrpc/connect-node'
import type { Interceptor, Transport } from '@connectrpc/connect'

// Generated service definitions (source of truth)
import { EmailService } from './gen/v1/email_pb'
import { DomainService } from './gen/v1/domain_pb'

// Re-export all proto types for SDK consumers
export * from './gen/v1/email_pb'
export * from './gen/v1/domain_pb'
export * from './gen/v1/events_pb'

/**
 * Configuration options for creating an Email API client.
 */
export interface ClientOptions {
    /**
     * API key for authentication.
     * Format: `em_...`
     */
    apiKey: string

    /**
     * Base URL of the Email API server.
     * @default "https://api.emailapi.dev"
     */
    baseUrl?: string

    /**
     * Custom transport for advanced use cases.
     * If provided, `baseUrl` and `apiKey` are ignored for transport creation.
     */
    transport?: Transport

    /**
     * Additional interceptors to apply to requests.
     */
    interceptors?: Interceptor[]
}

/**
 * Email API client interface.
 * Provides access to all Email API services.
 */
export interface EmailApiClient {
    /**
     * Email service client.
     * Use for sending emails and streaming events.
     */
    email: Client<typeof EmailService>

    /**
     * Domain service client.
     * Use for managing sending domains.
     */
    domains: Client<typeof DomainService>

    // Convenience shortcuts

    /**
     * Send an email.
     * Shortcut for `client.email.sendEmail()`.
     * 
     * @example
     * ```ts
     * const result = await client.send({
     *   from: 'hello@example.com',
     *   to: ['user@example.com'],
     *   subject: 'Hello!',
     *   body: 'World'
     * })
     * ```
     */
    send: Client<typeof EmailService>['sendEmail']

    /**
     * Stream events in real-time.
     * Shortcut for `client.email.streamEvents()`.
     * 
     * @example
     * ```ts
     * for await (const event of client.onReceive({ cursor: '0' })) {
     *   console.log(event.type, event.payload)
     * }
     * ```
     */
    onReceive: Client<typeof EmailService>['streamEvents']
}

/**
 * Create an Email API client.
 * 
 * @param options - Client configuration options
 * @returns A fully typed Email API client
 * 
 * @example
 * ```ts
 * import { createClient, EventType, EmailStatus } from '@emailapi/sdk'
 * 
 * const client = createClient({ apiKey: 'em_...' })
 * 
 * // Send an email
 * const result = await client.send({
 *   from: 'hello@yourdomain.com',
 *   to: ['user@example.com'],
 *   subject: 'Hello!',
 *   body: 'World'
 * })
 * 
 * console.log(result.id)  // string
 * console.log(result.status === EmailStatus.SENT)  // boolean
 * 
 * // Stream events
 * for await (const event of client.onReceive({ cursor: '0' })) {
 *   if (event.type === EventType.EMAIL_REPLIED) {
 *     console.log('Reply from:', event.payload.value?.from)
 *   }
 * }
 * ```
 */
export function createClient(options: ClientOptions): EmailApiClient {
    const baseUrl = options.baseUrl ?? 'https://api.emailapi.dev'

    const authInterceptor: Interceptor = (next) => async (req) => {
        req.header.set('Authorization', `Bearer ${options.apiKey}`)
        return next(req)
    }

    const transport = options.transport ?? createGrpcTransport({
        baseUrl,
        interceptors: [authInterceptor, ...(options.interceptors ?? [])],
    })

    const email = createConnectClient(EmailService, transport)
    const domains = createConnectClient(DomainService, transport)

    return {
        email,
        domains,
        // Convenience shortcuts
        send: email.sendEmail.bind(email),
        onReceive: email.streamEvents.bind(email),
    }
}
