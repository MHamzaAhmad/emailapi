import 'dotenv/config'
import { serve } from '@hono/node-server'
import { Hono } from 'hono'
import { createClient } from 'simpleemailapi'

const app = new Hono()

const client = createClient({
  apiKey: process.env.SIMPLE_EMAIL_API_KEY!,
  baseUrl: "http://localhost:8080"
})

const controller = client.onReceive({
  onReplied(event) {
    console.log('Received reply:', event.messageId)
  },
  onError(error) {
    console.error('Error in event stream:', error)
  },
})

// Stop the stream when the server is stopped
process.on('SIGINT', () => {
  controller.abort()
})

serve({
  fetch: app.fetch,
  port: 3002
}, (info) => {
  console.log(`Server is running on http://localhost:${info.port}`)
})
