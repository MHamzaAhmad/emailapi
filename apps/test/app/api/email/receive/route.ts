import { createClient } from "simpleemailapi"

export async function GET() {
    const client = createClient({
        apiKey: process.env.SIMPLE_EMAIL_API_KEY!,
    })

    const stream = new ReadableStream({
        start(controller) {
            client.onReceive({
                onReplied(event) {
                    controller.enqueue(JSON.stringify(event))
                },
            })
        }
    })

    return new Response(stream, {
        headers: {
            "Content-Type": "text/event-stream",
        },
    })
}