import { NextResponse, NextRequest } from "next/server";
import { createClient, SendEmailRequest } from "simpleemailapi"

export async function POST(request: NextRequest) {
    try {
        const client = createClient({
            apiKey: process.env.SIMPLE_EMAIL_API_KEY!,
        })

        const body = await request.json() as SendEmailRequest

        const res = await client.send({
            from: body.from,
            to: body.to,
            subject: body.subject,
            body: body.body,
            async: body.async
        })

        return NextResponse.json({ message: "Email sent successfully" }, { status: 200 });
    } catch (error) {
        return NextResponse.json({ error: "Failed to send email" }, { status: 500 });
    }
}