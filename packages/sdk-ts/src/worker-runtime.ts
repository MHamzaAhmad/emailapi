/**
 * Worker Runtime for onReceive
 * 
 * This runs in a separate worker thread to keep the main event loop responsive.
 * It handles:
 * - gRPC streaming connection
 * - Auto-reconnection with exponential backoff
 * - Cursor tracking for resume
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
    cursor: string
    batchSize: number
}

interface WorkerMessage {
    type: 'event' | 'connected' | 'disconnected' | 'error'
    payload?: unknown
    cursor?: string
}

const config = workerData as WorkerConfig

// Reconnection settings
const INITIAL_DELAY = 1000
const MAX_DELAY = 30000
const BACKOFF_MULTIPLIER = 2

let currentDelay = INITIAL_DELAY
let lastCursor = config.cursor
let shouldStop = false

// Listen for abort signal from parent
parentPort?.on('message', (msg: { type: string }) => {
    if (msg.type === 'abort') {
        shouldStop = true
    }
})

function post(message: WorkerMessage) {
    parentPort?.postMessage(message)
}

async function sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms))
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

    const email = createConnectClient(EmailService, transport)

    while (!shouldStop) {
        try {
            post({ type: 'connected' })
            currentDelay = INITIAL_DELAY // Reset backoff on successful connect

            const stream = email.streamEvents({
                cursor: lastCursor,
                eventTypes: [],
                batchSize: config.batchSize,
            })

            for await (const event of stream) {
                if (shouldStop) break

                // Update cursor for resume
                if (event.id) {
                    lastCursor = event.id
                }

                // Skip heartbeats
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
                        cursor: lastCursor,
                    })
                }
            }

            // Stream ended normally
            if (!shouldStop) {
                post({ type: 'disconnected' })
            }
        } catch (err) {
            if (shouldStop) break

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
}

// Start streaming
startStreaming().catch(err => {
    post({ type: 'error', payload: String(err) })
})
