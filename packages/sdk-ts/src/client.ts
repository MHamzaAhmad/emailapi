/**
 * Email API TypeScript SDK
 * 
 * Backend-first HTTP/2 gRPC client for Node.js 18+ and Bun.
 * Uses Connect RPC for type-safe communication.
 * 
 * @example
 * ```ts
 * import { createClient } from '@emailapi/sdk'
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
 * // Stream events with typed callbacks
 * const controller = client.onReceive({
 *   onDelivered: (event) => console.log('Delivered to:', event.recipients),
 *   onReplied: (event) => console.log('Reply from:', event.from),
 *   onBounced: (event) => console.log('Bounced:', event.bounceType),
 * })
 * 
 * // Later: stop listening
 * controller.abort()
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
import {
    EventType,
    type EmailSentEvent,
    type EmailDeliveredEvent,
    type EmailBouncedEvent,
    type EmailComplainedEvent,
    type EmailRejectedEvent,
    type EmailDelayedEvent,
    type EmailRepliedEvent,
    type EmailFailedEvent,
} from './gen/v1/events_pb'

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
 * Options for the `onReceive` event stream.
 * Define only the callbacks you care about - no need to handle every event type.
 */
export interface EventHandlers {
    /**
     * Starting cursor position for the stream.
     * Use '0' to start from the beginning, or a previous event ID to resume.
     * @default '0'
     */
    cursor?: string

    /**
     * Number of events to buffer per batch.
     * @default 10
     */
    batchSize?: number

    /**
     * Called when an email is accepted for delivery.
     */
    onSent?: (event: EmailSentEvent) => void

    /**
     * Called when an email is successfully delivered to the recipient's mailbox.
     */
    onDelivered?: (event: EmailDeliveredEvent) => void

    /**
     * Called when an email bounces (hard or soft bounce).
     */
    onBounced?: (event: EmailBouncedEvent) => void

    /**
     * Called when a recipient marks the email as spam.
     */
    onComplained?: (event: EmailComplainedEvent) => void

    /**
     * Called when SES rejects the email before sending.
     */
    onRejected?: (event: EmailRejectedEvent) => void

    /**
     * Called when email delivery is delayed.
     */
    onDelayed?: (event: EmailDelayedEvent) => void

    /**
     * Called when a reply to a sent email is received.
     */
    onReplied?: (event: EmailRepliedEvent) => void

    /**
     * Called when email sending fails permanently.
     */
    onFailed?: (event: EmailFailedEvent) => void

    /**
     * Called when an error occurs in the stream.
     */
    onError?: (error: Error) => void
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
     * Stream events in real-time with typed callbacks.
     * Define only the event handlers you need.
     * 
     * @returns AbortController to stop the stream
     * 
     * @example
     * ```ts
     * const controller = client.onReceive({
     *   cursor: '0',
     *   onDelivered: (event) => {
     *     console.log('Delivered to:', event.recipients)
     *   },
     *   onReplied: (event) => {
     *     console.log('Reply from:', event.from)
     *     console.log('Message:', event.body)
     *   },
     *   onBounced: (event) => {
     *     console.log('Bounced:', event.bounceType, event.recipients)
     *   },
     *   onError: (err) => {
     *     console.error('Stream error:', err)
     *   }
     * })
     * 
     * // Stop listening when done
     * controller.abort()
     * ```
     */
    onReceive: (handlers: EventHandlers) => AbortController
}

/**
 * Create an Email API client.
 * 
 * @param options - Client configuration options
 * @returns A fully typed Email API client
 * 
 * @example
 * ```ts
 * import { createClient } from '@emailapi/sdk'
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
 * // Stream events with typed handlers
 * const controller = client.onReceive({
 *   onReplied: (event) => {
 *     console.log('Got reply from:', event.from)
 *     console.log('Subject:', event.subject)
 *     console.log('Body:', event.body)
 *   },
 *   onBounced: (event) => {
 *     console.log('Email bounced:', event.bounceType)
 *   },
 *   onError: (err) => console.error(err)
 * })
 * 
 * // Later: stop the stream
 * controller.abort()
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

    function onReceive(handlers: EventHandlers): AbortController {
        const controller = new AbortController()

            // Start streaming in the background
            ; (async () => {
                try {
                    const stream = email.streamEvents(
                        {
                            cursor: handlers.cursor ?? '0',
                            eventTypes: [],
                            batchSize: handlers.batchSize ?? 10,
                        },
                        { signal: controller.signal }
                    )

                    for await (const event of stream) {
                        // Skip heartbeat events
                        if (event.type === EventType.HEARTBEAT) {
                            continue
                        }

                        // Dispatch to the appropriate handler based on payload type
                        const payload = event.payload
                        if (!payload || payload.case === undefined) {
                            continue
                        }

                        switch (payload.case) {
                            case 'emailSent':
                                handlers.onSent?.(payload.value)
                                break
                            case 'emailDelivered':
                                handlers.onDelivered?.(payload.value)
                                break
                            case 'emailBounced':
                                handlers.onBounced?.(payload.value)
                                break
                            case 'emailComplained':
                                handlers.onComplained?.(payload.value)
                                break
                            case 'emailRejected':
                                handlers.onRejected?.(payload.value)
                                break
                            case 'emailDelayed':
                                handlers.onDelayed?.(payload.value)
                                break
                            case 'emailReplied':
                                handlers.onReplied?.(payload.value)
                                break
                            case 'emailFailed':
                                handlers.onFailed?.(payload.value)
                                break
                        }
                    }
                } catch (err) {
                    // Don't report abort errors
                    if (err instanceof Error && err.name === 'AbortError') {
                        return
                    }
                    handlers.onError?.(err instanceof Error ? err : new Error(String(err)))
                }
            })()

        return controller
    }

    return {
        email,
        domains,
        send: email.sendEmail.bind(email),
        onReceive,
    }
}

