// Email types
export type EmailStatus = 'pending' | 'sent' | 'delivered' | 'failed' | 'bounced'

export interface SendEmailRequest {
    from: string
    to: string[]
    cc?: string[]
    bcc?: string[]
    subject: string
    body?: string
    html?: string
    metadata?: Record<string, unknown>
    scheduled_at?: string
}

export interface SendEmailResponse {
    id: string
    status: EmailStatus
}

export interface Email {
    id: string
    from: string
    to: string[]
    cc?: string[]
    bcc?: string[]
    subject: string
    body?: string
    html?: string
    status: EmailStatus
    provider_id?: string
    user_id: string
    metadata?: Record<string, unknown>
    scheduled_at?: string
    sent_at?: string
    created_at: string
    updated_at: string
}

export interface EmailListResponse {
    data: Email[]
    limit: number
    offset: number
}

// Webhook types
export type WebhookEventType =
    | 'email.sent'
    | 'email.delivered'
    | 'email.failed'
    | 'email.bounced'
    | 'email.opened'
    | 'email.clicked'

export interface CreateWebhookRequest {
    name: string
    url: string
    events: WebhookEventType[]
}

export interface UpdateWebhookRequest {
    name?: string
    url?: string
    events?: WebhookEventType[]
    is_active?: boolean
}

export interface Webhook {
    id: string
    user_id: string
    name: string
    url: string
    events: WebhookEventType[]
    is_active: boolean
    retry_count: number
    last_success?: string
    last_failure?: string
    created_at: string
    updated_at: string
}

export interface WebhookWithSecret {
    webhook: Webhook
    secret: string
    message: string
}

export interface WebhookListResponse {
    data: Webhook[]
}

// User types
export type UserRole = 'admin' | 'member'

export interface User {
    id: string
    email: string
    name: string
    role: UserRole
    api_key_prefix?: string
    is_active: boolean
    created_at: string
    updated_at: string
}

// Error types
export interface ApiError {
    error: string
    details?: string
}
