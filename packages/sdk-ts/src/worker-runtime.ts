/**
 * Worker Runtime for onReceive
 * 
 * This runs in a separate worker thread to keep the main event loop responsive.
 * It handles:
 * - gRPC streaming connection with Redis Consumer Groups
 * - Auto-reconnection with exponential backoff
 * - Automatic event acknowledgment (when ackMode is 'auto')
 * - Batched acks for efficiency
 */

import { parentPort, workerData } from 'worker_threads'
import { createClient as createConnectClient } from '@connectrpc/connect'
import { createGrpcTransport } from '@connectrpc/connect-node'
import type { Interceptor } from '@connectrpc/connect'

import { EmailService } from './gen/v1/email_pb'
import { EventType } from './gen/v1/events_pb'

interface WorkerConfig {
    apiKey: string
    baseUrl: string
    batchSize: number
    ackMode: 'auto' | 'manual'
}

interface WorkerMessage {
    type: 'event' | 'connected' | 'disconnected' | 'error'
    payload?: unknown
    eventId?: string
}

const config = workerData as WorkerConfig

// Reconnection settings
const INITIAL_DELAY = 1000
const MAX_DELAY = 30000
const BACKOFF_MULTIPLIER = 2

// Ack batching settings
const ACK_BATCH_SIZE = 10
const ACK_FLUSH_INTERVAL_MS = 1000

let currentDelay = INITIAL_DELAY
let shouldStop = false

// Pending event IDs to acknowledge
let pendingAcks: string[] = []
let ackFlushTimer: ReturnType<typeof setTimeout> | null = null

// Listen for abort signal from parent
parentPort?.on('message', (msg: { type: string; eventId?: string }) => {
    if (msg.type === 'abort') {
        shouldStop = true
        // Flush any pending acks before stopping
        flushAcks()
    } else if (msg.type === 'ack' && msg.eventId) {
        // Parent confirmed event was handled successfully
        queueAck(msg.eventId)
    }
})

function post(message: WorkerMessage) {
    parentPort?.postMessage(message)
}

async function sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms))
}

// Email service reference for acking
let emailService: ReturnType<typeof createConnectClient<typeof EmailService>> | null = null

async function flushAcks(): Promise<void> {
    if (pendingAcks.length === 0 || !emailService) return

    const idsToAck = [...pendingAcks]
    pendingAcks = []

    if (ackFlushTimer) {
        clearTimeout(ackFlushTimer)
        ackFlushTimer = null
    }

    try {
        await emailService.ackEvents({ eventIds: idsToAck })
    } catch (err) {
        // Log but don't fail - server will replay on next connect
        console.error('Failed to ack events:', err)
    }
}

function scheduleAckFlush(): void {
    if (ackFlushTimer) return // Already scheduled

    ackFlushTimer = setTimeout(async () => {
        ackFlushTimer = null
        await flushAcks()
    }, ACK_FLUSH_INTERVAL_MS)
}

function queueAck(eventId: string): void {
    if (config.ackMode !== 'auto') return

    pendingAcks.push(eventId)

    // Flush immediately if batch is full
    if (pendingAcks.length >= ACK_BATCH_SIZE) {
        // Don't await - let it run in background
        flushAcks().catch(err => console.error('Flush ack error:', err))
    } else {
        // Otherwise schedule a flush
        scheduleAckFlush()
    }
}

async function startStreaming(): Promise<void> {
    const authInterceptor: Interceptor = (next) => async (req) => {
        req.header.set('Authorization', `Bearer ${config.apiKey}`)
        return next(req)
    }

    const transport = createGrpcTransport({
        baseUrl: config.baseUrl,
        interceptors: [authInterceptor],
    })

    emailService = createConnectClient(EmailService, transport)

    while (!shouldStop) {
        try {
            post({ type: 'connected' })
            currentDelay = INITIAL_DELAY // Reset backoff on successful connect

            const stream = emailService.streamEvents({
                eventTypes: [],
                batchSize: config.batchSize,
            })

            for await (const event of stream) {
                if (shouldStop) break

                // Skip heartbeats (don't need to ack these)
                if (event.type === EventType.HEARTBEAT) {
                    continue
                }

                // Send event to parent
                const payload = event.payload
                if (payload && payload.case !== undefined) {
                    post({
                        type: 'event',
                        payload: {
                            case: payload.case,
                            value: payload.value,
                        },
                        eventId: event.id, // Include event ID for acking after handler completes
                    })
                }
            }

            // Stream ended normally - flush remaining acks
            await flushAcks()

            if (!shouldStop) {
                post({ type: 'disconnected' })
            }
        } catch (err) {
            if (shouldStop) break

            // Try to flush acks before reconnecting
            await flushAcks()

            post({
                type: 'error',
                payload: err instanceof Error ? err.message : String(err)
            })
            post({ type: 'disconnected' })

            // Exponential backoff
            await sleep(currentDelay)
            currentDelay = Math.min(currentDelay * BACKOFF_MULTIPLIER, MAX_DELAY)
        }
    }

    // Final flush before exit
    await flushAcks()
}

// Start streaming
startStreaming().catch(err => {
    post({ type: 'error', payload: String(err) })
})
