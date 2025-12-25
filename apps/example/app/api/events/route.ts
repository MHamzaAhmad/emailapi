import { createClient, EventType, type Event } from '@emailapi/sdk'

export const runtime = 'nodejs'
export const dynamic = 'force-dynamic'

// Extract payload data from event, converting to plain objects
function extractPayloadData(event: Event): Record<string, unknown> {
    const base = {
        id: event.id,
        type: getEventTypeName(event.type),
    }

    const payload = event.payload
    if (!payload || payload.case === undefined) {
        return base
    }

    const value = payload.value
    if (!value) {
        return base
    }

    const extracted: Record<string, unknown> = { ...base }

    // Type-safe extraction of common fields
    if ('emailId' in value) extracted.emailId = value.emailId
    if ('userId' in value) extracted.userId = value.userId
    if ('messageId' in value) extracted.messageId = value.messageId
    if ('from' in value) extracted.from = value.from
    if ('to' in value) extracted.to = value.to
    if ('subject' in value) extracted.subject = value.subject
    if ('recipients' in value) extracted.recipients = value.recipients
    if ('error' in value) extracted.error = value.error
    if ('bounceType' in value) extracted.bounceType = value.bounceType
    if ('bounceSubtype' in value) extracted.bounceSubtype = value.bounceSubtype
    if ('feedbackType' in value) extracted.feedbackType = value.feedbackType
    if ('reason' in value) extracted.reason = value.reason
    if ('delayType' in value) extracted.delayType = value.delayType

    return extracted
}

function getEventTypeName(type: EventType): string {
    const names: Record<EventType, string> = {
        [EventType.UNSPECIFIED]: 'unspecified',
        [EventType.EMAIL_SENT]: 'email.sent',
        [EventType.EMAIL_DELIVERED]: 'email.delivered',
        [EventType.EMAIL_FAILED]: 'email.failed',
        [EventType.EMAIL_BOUNCED]: 'email.bounced',
        [EventType.EMAIL_OPENED]: 'email.opened',
        [EventType.EMAIL_CLICKED]: 'email.clicked',
        [EventType.EMAIL_REPLIED]: 'email.replied',
        [EventType.EMAIL_COMPLAINED]: 'email.complained',
        [EventType.EMAIL_REJECTED]: 'email.rejected',
        [EventType.EMAIL_DELAYED]: 'email.delayed',
    }
    return names[type] ?? 'unknown'
}

export async function GET(request: Request) {
    const { searchParams } = new URL(request.url)
    const cursor = searchParams.get('cursor') ?? '0'

    const apiKey = process.env.EMAILAPI_KEY
    if (!apiKey) {
        return new Response('API key not configured', { status: 500 })
    }

    const client = createClient({
        apiKey,
        baseUrl: process.env.EMAILAPI_URL ?? 'http://localhost:8080',
    })

    // Create a TransformStream to stream events as Server-Sent Events
    const stream = new TransformStream()
    const writer = stream.writable.getWriter()
    const encoder = new TextEncoder()

    // Start streaming in the background
    let heartbeatInterval: NodeJS.Timeout | null = null

        ; (async () => {
            try {
                console.log('[Events API] Starting event stream from cursor:', cursor)

                // Send a heartbeat comment every 15 seconds to keep connection alive
                heartbeatInterval = setInterval(() => {
                    try {
                        writer.write(encoder.encode(': heartbeat\n\n'))
                    } catch (err) {
                        console.error('[Events API] Failed to send heartbeat:', err)
                    }
                }, 15000)

                // Stream events from backend using the raw email service
                const eventStream = client.email.streamEvents({ cursor, eventTypes: [], batchSize: 10 })

                for await (const event of eventStream) {
                    // Skip heartbeat events
                    if (event.type === EventType.HEARTBEAT) {
                        continue
                    }

                    const payloadData = extractPayloadData(event)

                    const eventData = {
                        id: event.id,
                        type: getEventTypeName(event.type),
                        timestamp: event.timestamp
                            ? new Date(Number(event.timestamp.seconds) * 1000).toISOString()
                            : new Date().toISOString(),
                        data: payloadData,
                        cursor: event.id,
                    }

                    console.log('[Events API] Received event:', eventData.type, eventData.id)

                    // Write as Server-Sent Event
                    await writer.write(
                        encoder.encode(`data: ${JSON.stringify(eventData)}\n\n`)
                    )
                }
            } catch (error) {
                console.error('[Events API] Stream error:', error)
                try {
                    await writer.write(
                        encoder.encode(`data: ${JSON.stringify({ error: error instanceof Error ? error.message : 'Stream error' })}\n\n`)
                    )
                } catch (writeErr) {
                    console.error('[Events API] Failed to write error:', writeErr)
                }
            } finally {
                if (heartbeatInterval) {
                    clearInterval(heartbeatInterval)
                }
                console.log('[Events API] Stream ended')
                try {
                    await writer.close()
                } catch (closeErr) {
                    console.error('[Events API] Failed to close writer:', closeErr)
                }
            }
        })()

    return new Response(stream.readable, {
        headers: {
            'Content-Type': 'text/event-stream',
            'Cache-Control': 'no-cache',
            'Connection': 'keep-alive',
            'X-Accel-Buffering': 'no', // Disable proxy buffering
        },
    })
}
