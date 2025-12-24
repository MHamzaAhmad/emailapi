'use client'

import { useState, useTransition, useEffect, useRef, useCallback } from 'react'
import { sendEmail, type SendEmailInput, type SendEmailResult, type EventData } from './actions'

export default function Home() {
  const [isPending, startTransition] = useTransition()
  const [result, setResult] = useState<SendEmailResult | null>(null)
  const [events, setEvents] = useState<EventData[]>([])
  const [isStreaming, setIsStreaming] = useState(false)
  const [streamError, setStreamError] = useState<string | null>(null)
  const [cursor, setCursor] = useState('0')
  const eventSourceRef = useRef<EventSource | null>(null)

  const [form, setForm] = useState<SendEmailInput>({
    from: '',
    to: '',
    subject: 'Test Email from SDK Example',
    body: 'Hello! This is a test email sent using the @emailapi/sdk.',
    async: false,
  })

  const startStreaming = useCallback(async () => {
    if (isStreaming) return

    setIsStreaming(true)
    setStreamError(null)

    try {
      // Use EventSource to consume the streaming API route
      const eventSource = new EventSource(`/api/events?cursor=${cursor}`)

      eventSource.onmessage = (event) => {
        try {
          const eventData = JSON.parse(event.data)

          // Check for error
          if (eventData.error) {
            setStreamError(eventData.error)
            eventSource.close()
            setIsStreaming(false)
            return
          }

          setEvents((prev) => {
            // Avoid duplicates
            if (prev.some(e => e.id === eventData.id)) return prev
            return [{
              id: eventData.id,
              type: eventData.type,
              timestamp: eventData.timestamp,
              data: JSON.stringify(eventData.data, null, 2),
              cursor: eventData.cursor,
            }, ...prev].slice(0, 100) // Keep last 100 events
          })
          setCursor(eventData.cursor)
        } catch (err) {
          console.error('Failed to parse event:', err)
        }
      }

      eventSource.onerror = (err) => {
        console.error('EventSource error:', err)
        eventSource.close()
        setStreamError('Connection lost')
        setIsStreaming(false)
      }

      // Store eventSource in a ref or state so we can close it
      eventSourceRef.current = eventSource
    } catch (error) {
      setStreamError(error instanceof Error ? error.message : 'Stream error')
      setIsStreaming(false)
    }
  }, [isStreaming, cursor])

  const stopStreaming = useCallback(() => {
    eventSourceRef.current?.close()
    setIsStreaming(false)
  }, [])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    startTransition(async () => {
      const res = await sendEmail(form)
      setResult(res)
      if (res.success && res.data) {
        // Add to events list locally (server stream will also pick it up)
        setEvents((prev) => [
          {
            id: res.data!.id,
            type: 'email.sent',
            timestamp: new Date().toISOString(),
            data: JSON.stringify(res.data, null, 2),
            cursor: res.data!.id,
          },
          ...prev,
        ])
      }
    })
  }

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100">
      <div className="max-w-6xl mx-auto p-8">
        {/* Header */}
        <header className="mb-12">
          <h1 className="text-4xl font-bold bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">
            @emailapi/sdk Example
          </h1>
          <p className="text-gray-400 mt-2">
            Test the Email API SDK with real email sending and event streaming.
          </p>
        </header>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Send Email Form */}
          <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-6">
            <h2 className="text-xl font-semibold mb-6 flex items-center gap-2">
              <span className="w-2 h-2 bg-blue-500 rounded-full"></span>
              Send Email
            </h2>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm text-gray-400 mb-1">From</label>
                <input
                  type="email"
                  value={form.from}
                  onChange={(e) => setForm({ ...form, from: e.target.value })}
                  placeholder="sender@yourdomain.com"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 focus:border-blue-500 focus:outline-none"
                  required
                />
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">To</label>
                <input
                  type="email"
                  value={form.to}
                  onChange={(e) => setForm({ ...form, to: e.target.value })}
                  placeholder="recipient@example.com"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 focus:border-blue-500 focus:outline-none"
                  required
                />
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Subject</label>
                <input
                  type="text"
                  value={form.subject}
                  onChange={(e) => setForm({ ...form, subject: e.target.value })}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 focus:border-blue-500 focus:outline-none"
                  required
                />
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Body</label>
                <textarea
                  value={form.body}
                  onChange={(e) => setForm({ ...form, body: e.target.value })}
                  rows={4}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 focus:border-blue-500 focus:outline-none resize-none"
                  required
                />
              </div>

              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="async"
                  checked={form.async}
                  onChange={(e) => setForm({ ...form, async: e.target.checked })}
                  className="rounded bg-gray-800 border-gray-700"
                />
                <label htmlFor="async" className="text-sm text-gray-400">
                  Async (queue with retries)
                </label>
              </div>

              <button
                type="submit"
                disabled={isPending}
                className="w-full bg-blue-600 hover:bg-blue-500 disabled:bg-gray-700 disabled:cursor-not-allowed text-white font-medium py-2 px-4 rounded-lg transition-colors"
              >
                {isPending ? 'Sending...' : 'Send Email'}
              </button>
            </form>

            {/* Result */}
            {result && (
              <div
                className={`mt-6 p-4 rounded-lg ${result.success ? 'bg-green-900/30 border border-green-800' : 'bg-red-900/30 border border-red-800'
                  }`}
              >
                <h3 className={`font-medium ${result.success ? 'text-green-400' : 'text-red-400'}`}>
                  {result.success ? '✓ Email Sent' : '✗ Error'}
                </h3>
                <pre className="mt-2 text-sm text-gray-300 overflow-auto">
                  {result.success ? JSON.stringify(result.data, null, 2) : result.error}
                </pre>
              </div>
            )}
          </div>

          {/* Events Panel */}
          <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-6">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-semibold flex items-center gap-2">
                <span className={`w-2 h-2 rounded-full ${isStreaming ? 'bg-green-500 animate-pulse' : 'bg-gray-500'}`}></span>
                Events
              </h2>
              <div className="flex items-center gap-2">
                {!isStreaming ? (
                  <button
                    onClick={startStreaming}
                    className="px-3 py-1 text-sm bg-purple-600 hover:bg-purple-500 text-white rounded-lg transition-colors"
                  >
                    Start Listening
                  </button>
                ) : (
                  <button
                    onClick={stopStreaming}
                    className="px-3 py-1 text-sm bg-red-600 hover:bg-red-500 text-white rounded-lg transition-colors"
                  >
                    Stop
                  </button>
                )}
                <button
                  onClick={() => setEvents([])}
                  className="px-3 py-1 text-sm bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors"
                >
                  Clear
                </button>
              </div>
            </div>

            {streamError && (
              <div className="mb-4 p-3 bg-red-900/30 border border-red-800 rounded-lg text-sm text-red-400">
                {streamError}
              </div>
            )}

            {events.length === 0 ? (
              <div className="text-center py-12 text-gray-500">
                <p>No events yet.</p>
                <p className="text-sm mt-1">
                  {isStreaming ? 'Listening for events...' : 'Click "Start Listening" and send an email.'}
                </p>
              </div>
            ) : (
              <div className="space-y-4 max-h-[600px] overflow-auto">
                {events.map((event, i) => (
                  <div key={`${event.id}-${i}`} className="bg-gray-800/50 border border-gray-700 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-sm font-mono text-purple-400">{event.type}</span>
                      <span className="text-xs text-gray-500">{new Date(event.timestamp).toLocaleTimeString()}</span>
                    </div>
                    <pre className="text-xs text-gray-400 overflow-auto">{event.data}</pre>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* SDK Usage Example */}
        <div className="mt-12 bg-gray-900/50 border border-gray-800 rounded-xl p-6">
          <h2 className="text-xl font-semibold mb-4">SDK Usage</h2>
          <pre className="bg-gray-800 rounded-lg p-4 overflow-auto text-sm">
            <code className="text-gray-300">{`import { createClient, EventType } from '@emailapi/sdk'

const client = createClient({ apiKey: 'em_...' })

// Send an email
const result = await client.send({
  from: 'hello@yourdomain.com',
  to: ['user@example.com'],
  subject: 'Hello!',
  body: 'World'
})

console.log(result.id, result.status)

// Stream events in real-time
for await (const event of client.onReceive({ cursor: '0' })) {
  if (event.type === EventType.EMAIL_DELIVERED) {
    console.log('Delivered!', event.payload)
  }
}`}</code>
          </pre>
        </div>
      </div>
    </div>
  )
}
