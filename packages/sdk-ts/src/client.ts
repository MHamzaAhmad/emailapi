import type {
    SendEmailRequest,
    SendEmailResponse,
    Email,
    EmailListResponse,
    Webhook,
    WebhookWithSecret,
    CreateWebhookRequest,
    UpdateWebhookRequest,
    WebhookListResponse,
    User,
} from './types'

export interface EmailApiClientConfig {
    /** API key in format "em_..." */
    apiKey: string
    /** Base URL of the API. Defaults to production. */
    baseUrl?: string
    /** Request timeout in milliseconds. Defaults to 30000. */
    timeout?: number
}

/**
 * Email API Client
 *
 * @example
 * ```ts
 * const client = new EmailApiClient({ apiKey: 'em_...' })
 *
 * const { id } = await client.emails.send({
 *   from: 'sender@example.com',
 *   to: ['recipient@example.com'],
 *   subject: 'Hello',
 *   body: 'World',
 * })
 * ```
 */
export class EmailApiClient {
    private readonly apiKey: string
    private readonly baseUrl: string
    private readonly timeout: number

    public readonly emails: EmailsApi
    public readonly webhooks: WebhooksApi
    public readonly users: UsersApi

    constructor(config: EmailApiClientConfig) {
        if (!config.apiKey) {
            throw new Error('API key is required')
        }

        this.apiKey = config.apiKey
        this.baseUrl = config.baseUrl ?? 'https://api.emailapi.dev/v1'
        this.timeout = config.timeout ?? 30000

        // Initialize API namespaces
        this.emails = new EmailsApi(this)
        this.webhooks = new WebhooksApi(this)
        this.users = new UsersApi(this)
    }

    /**
     * Make an authenticated request to the API.
     */
    async request<T>(
        method: string,
        path: string,
        body?: unknown
    ): Promise<T> {
        const url = `${this.baseUrl}${path}`

        const headers: Record<string, string> = {
            'Authorization': `Bearer ${this.apiKey}`,
            'Content-Type': 'application/json',
        }

        const controller = new AbortController()
        const timeoutId = setTimeout(() => controller.abort(), this.timeout)

        try {
            const response = await fetch(url, {
                method,
                headers,
                body: body ? JSON.stringify(body) : undefined,
                signal: controller.signal,
            })

            clearTimeout(timeoutId)

            if (!response.ok) {
                const error = await response.json().catch(() => ({}))
                throw new EmailApiError(
                    error.error || `Request failed with status ${response.status}`,
                    response.status,
                    error
                )
            }

            if (response.status === 204) {
                return undefined as T
            }

            return await response.json()
        } catch (error) {
            clearTimeout(timeoutId)
            if (error instanceof EmailApiError) {
                throw error
            }
            throw new EmailApiError(
                error instanceof Error ? error.message : 'Unknown error',
                0
            )
        }
    }
}

/**
 * Email API Error
 */
export class EmailApiError extends Error {
    constructor(
        message: string,
        public readonly status: number,
        public readonly response?: unknown
    ) {
        super(message)
        this.name = 'EmailApiError'
    }
}

/**
 * Emails API namespace
 */
class EmailsApi {
    constructor(private readonly client: EmailApiClient) { }

    /**
     * Send an email
     */
    async send(request: SendEmailRequest): Promise<SendEmailResponse> {
        return this.client.request<SendEmailResponse>('POST', '/send', request)
    }

    /**
     * Get an email by ID
     */
    async get(id: string): Promise<Email> {
        return this.client.request<Email>('GET', `/emails/${id}`)
    }

    /**
     * List emails with pagination
     */
    async list(options?: { limit?: number; offset?: number }): Promise<EmailListResponse> {
        const params = new URLSearchParams()
        if (options?.limit) params.set('limit', options.limit.toString())
        if (options?.offset) params.set('offset', options.offset.toString())
        const query = params.toString()
        return this.client.request<EmailListResponse>('GET', `/emails${query ? `?${query}` : ''}`)
    }
}

/**
 * Webhooks API namespace
 */
class WebhooksApi {
    constructor(private readonly client: EmailApiClient) { }

    /**
     * Create a webhook
     */
    async create(request: CreateWebhookRequest): Promise<WebhookWithSecret> {
        return this.client.request<WebhookWithSecret>('POST', '/webhooks', request)
    }

    /**
     * Get a webhook by ID
     */
    async get(id: string): Promise<Webhook> {
        return this.client.request<Webhook>('GET', `/webhooks/${id}`)
    }

    /**
     * List all webhooks
     */
    async list(): Promise<WebhookListResponse> {
        return this.client.request<WebhookListResponse>('GET', '/webhooks')
    }

    /**
     * Update a webhook
     */
    async update(id: string, request: UpdateWebhookRequest): Promise<Webhook> {
        return this.client.request<Webhook>('PATCH', `/webhooks/${id}`, request)
    }

    /**
     * Delete a webhook
     */
    async delete(id: string): Promise<void> {
        return this.client.request<void>('DELETE', `/webhooks/${id}`)
    }
}

/**
 * Users API namespace
 */
class UsersApi {
    constructor(private readonly client: EmailApiClient) { }

    /**
     * Get current user
     */
    async me(): Promise<User> {
        return this.client.request<User>('GET', '/users/me')
    }

    /**
     * Regenerate API key
     */
    async regenerateApiKey(): Promise<{ api_key: string; message: string }> {
        return this.client.request('POST', '/users/me/api-key')
    }
}
