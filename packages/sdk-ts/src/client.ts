/**
 * Email API TypeScript SDK
 * 
 * Backend-first HTTP/2 gRPC client for Node.js 18+ and Bun.
 * Uses Connect RPC for type-safe communication.
 * 
 * @example
 * ```ts
 * import { createClient } from 'simpleemailapi'
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
 * // Stream events - runs in background worker thread
 * // Events are automatically acknowledged after your handler completes
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
import { Worker } from 'worker_threads'
import { fileURLToPath } from 'url'
import { dirname, join } from 'path'
import type { Interceptor, Transport } from '@connectrpc/connect'

// Generated service definitions (source of truth)
import { EmailService } from './gen/v1/email_pb'
import { DomainService } from './gen/v1/domain_pb'
import type {
    EmailSentEvent,
    EmailDeliveredEvent,
    EmailBouncedEvent,
    EmailComplainedEvent,
    EmailRejectedEvent,
    EmailDelayedEvent,
    EmailRepliedEvent,
    EmailFailedEvent,
} from './gen/v1/events_pb'

// Re-export all proto types for SDK consumers
export * from './gen/v1/email_pb'
export * from './gen/v1/domain_pb'
export * from './gen/v1/events_pb'

// Error handling utilities
export { ErrorCode, SimpleEmailError, parseError, isSimpleEmailError } from './errors'

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
     * @default "https://api.simpleemailapi.dev"
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
 * 
 * Behind the scenes, onReceive:
 * - Runs in a worker thread (non-blocking)
 * - Auto-reconnects on disconnect
 * - Server-side tracking ensures you never miss or double-process events
 * - Events are automatically acknowledged after your handler completes
 */
export interface EventHandlers {
    /**
     * Acknowledgment mode.
     * - 'auto' (default): Events are automatically acknowledged after your handler completes
     * - 'manual': You must call ackEvents() to acknowledge processed events
     * @default 'auto'
     */
    ackMode?: 'auto' | 'manual'

    /**
     * Number of events to buffer per batch.
     * @default 10
     */
    batchSize?: number

    /** Called when an email is accepted for delivery. */
    onSent?: (event: EmailSentEvent) => void

    /** Called when an email is successfully delivered. */
    onDelivered?: (event: EmailDeliveredEvent) => void

    /** Called when an email bounces. */
    onBounced?: (event: EmailBouncedEvent) => void

    /** Called when a recipient marks the email as spam. */
    onComplained?: (event: EmailComplainedEvent) => void

    /** Called when SES rejects the email. */
    onRejected?: (event: EmailRejectedEvent) => void

    /** Called when email delivery is delayed. */
    onDelayed?: (event: EmailDelayedEvent) => void

    /** Called when a reply is received. */
    onReplied?: (event: EmailRepliedEvent) => void

    /** Called when email sending fails permanently. */
    onFailed?: (event: EmailFailedEvent) => void

    /** Called when an error occurs in the stream. */
    onError?: (error: Error) => void
}

/**
 * Email API client interface.
 */
export interface EmailApiClient {
    /** Email service client. */
    email: Client<typeof EmailService>

    /** Domain service client. */
    domains: Client<typeof DomainService>

    /**
     * Send an email.
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
     * 
     * Runs in a background worker thread for non-blocking operation.
     * Server-side tracking ensures you never miss or double-process events.
     * Events are automatically acknowledged after your handler completes.
     * 
     * @returns AbortController to stop the stream
     * 
     * @example
     * ```ts
     * const controller = client.onReceive({
     *   onReplied: (event) => console.log('Reply from:', event.from),
     *   onBounced: (event) => console.log('Bounced:', event.bounceType),
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
 */
export function createClient(options: ClientOptions): EmailApiClient {
    const baseUrl = options.baseUrl ?? 'https://api.simpleemailapi.dev'

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

    /**
     * onReceive - Worker thread backed event streaming
     * 
     * Spawns a worker thread to handle streaming, keeping main thread responsive.
     * Server tracks acknowledgment state using API key ID.
     */
    function onReceive(handlers: EventHandlers): AbortController {
        const controller = new AbortController()

        // Get path to worker runtime
        const __filename = fileURLToPath(import.meta.url)
        const __dirname = dirname(__filename)
        const workerPath = join(__dirname, 'worker-runtime.js')

        // Spawn worker with config
        const worker = new Worker(workerPath, {
            workerData: {
                apiKey: options.apiKey,
                baseUrl,
                batchSize: handlers.batchSize ?? 10,
                ackMode: handlers.ackMode ?? 'auto',
            },
        })

        // Handle messages from worker
        worker.on('message', (msg: { type: string; payload?: unknown; eventId?: string }) => {
            if (msg.type === 'event' && msg.payload) {
                const { case: eventCase, value } = msg.payload as { case: string; value: unknown }

                try {
                    switch (eventCase) {
                        case 'emailSent':
                            handlers.onSent?.(value as EmailSentEvent)
                            break
                        case 'emailDelivered':
                            handlers.onDelivered?.(value as EmailDeliveredEvent)
                            break
                        case 'emailBounced':
                            handlers.onBounced?.(value as EmailBouncedEvent)
                            break
                        case 'emailComplained':
                            handlers.onComplained?.(value as EmailComplainedEvent)
                            break
                        case 'emailRejected':
                            handlers.onRejected?.(value as EmailRejectedEvent)
                            break
                        case 'emailDelayed':
                            handlers.onDelayed?.(value as EmailDelayedEvent)
                            break
                        case 'emailReplied':
                            handlers.onReplied?.(value as EmailRepliedEvent)
                            break
                        case 'emailFailed':
                            handlers.onFailed?.(value as EmailFailedEvent)
                            break
                    }

                    // After handler completes successfully, tell worker to ack
                    if (msg.eventId && handlers.ackMode !== 'manual') {
                        worker.postMessage({ type: 'ack', eventId: msg.eventId })
                    }
                } catch (err) {
                    handlers.onError?.(err instanceof Error ? err : new Error(String(err)))
                    // Don't ack if handler threw an error - event will be replayed
                }
            } else if (msg.type === 'error') {
                handlers.onError?.(new Error(String(msg.payload)))
            }
        })

        worker.on('error', (err: Error) => {
            handlers.onError?.(err)
        })

        // Wire up abort controller
        controller.signal.addEventListener('abort', () => {
            worker.postMessage({ type: 'abort' })
            // Give worker time to clean up, then terminate
            setTimeout(() => worker.terminate(), 1000)
        })

        return controller
    }

    return {
        email,
        domains,
        send: email.sendEmail.bind(email),
        onReceive,
    }
}
