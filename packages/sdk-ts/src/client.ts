import { createClient as createConnectClient, type Client } from '@connectrpc/connect'
import { createGrpcWebTransport } from '@connectrpc/connect-web'

import { EmailService } from './generated/v1/email_pb'
import { UserService } from './generated/v1/user_pb'
import { WebhookService } from './generated/v1/webhook_pb'

export interface EmailApiClientConfig {
    /**
     * Base URL of the Email API server.
     * @default "https://api.emailapi.dev"
     */
    baseUrl?: string

    /**
     * API key for authentication (format: em_...)
     */
    apiKey: string
}

export interface EmailApiClient {
    emails: Client<typeof EmailService>
    users: Client<typeof UserService>
    webhooks: Client<typeof WebhookService>
}

/**
 * Creates a new Email API client.
 * 
 * @example
 * ```ts
 * import { createClient } from '@email-api/sdk-ts'
 * 
 * const client = createClient({ apiKey: 'em_...' })
 * 
 * const response = await client.emails.sendEmail({
 *   from: 'sender@example.com',
 *   to: ['recipient@example.com'],
 *   subject: 'Hello',
 *   body: 'World',
 * })
 * ```
 */
export function createClient(config: EmailApiClientConfig): EmailApiClient {
    const baseUrl = config.baseUrl ?? 'https://api.emailapi.dev'

    const transport = createGrpcWebTransport({
        baseUrl,
        interceptors: [
            (next) => async (req) => {
                req.header.set('Authorization', `Bearer ${config.apiKey}`)
                return next(req)
            },
        ],
    })

    return {
        emails: createConnectClient(EmailService, transport),
        users: createConnectClient(UserService, transport),
        webhooks: createConnectClient(WebhookService, transport),
    }
}
