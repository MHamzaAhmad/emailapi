'use server'

import { createClient, type SendEmailRequest, type SendEmailResponse, EventType, type Event } from '@emailapi/sdk'

// Create SDK client (uses environment variable for API key)
function getClient() {
    const apiKey = process.env.EMAILAPI_KEY
    if (!apiKey) {
        throw new Error('EMAILAPI_KEY environment variable is not set')
    }
    return createClient({
        apiKey,
        baseUrl: process.env.EMAILAPI_URL ?? 'http://localhost:8080',
    })
}

export type SendEmailInput = {
    from: string
    to: string
    subject: string
    body: string
    html?: string
    async?: boolean
}

export type SendEmailResult = {
    success: boolean
    data?: {
        id: string
        status: string
        messageId?: string
        statusMessage?: string
    }
    error?: string
}

/**
 * Send an email using the SDK
 */
export async function sendEmail(input: SendEmailInput): Promise<SendEmailResult> {
    try {
        const client = getClient()

        const response = await client.send({
            from: input.from,
            to: [input.to],
            subject: input.subject,
            body: input.body,
            html: input.html ?? '',
            async: input.async ?? false,
        })

        return {
            success: true,
            data: {
                id: response.id,
                status: getStatusName(response.status),
                messageId: response.messageId || undefined,
                statusMessage: response.statusMessage || undefined,
            },
        }
    } catch (error) {
        return {
            success: false,
            error: error instanceof Error ? error.message : 'Unknown error occurred',
        }
    }
}

// Map enum to string
function getStatusName(status: number): string {
    const statusNames: Record<number, string> = {
        0: 'unspecified',
        1: 'queued',
        2: 'processing',
        3: 'sent',
        4: 'failed',
    }
    return statusNames[status] ?? 'unknown'
}
