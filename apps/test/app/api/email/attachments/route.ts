import { NextResponse } from "next/server";
import { createClient, type SendEmailRequest } from "simpleemailapi";

// Input format for attachments from JSON body
interface AttachmentInput {
    filename: string;
    contentType: string;
    url?: string;
    base64?: string;
}

// Request body with our custom attachment format
interface EmailRequestBody extends Omit<SendEmailRequest, 'attachments'> {
    attachments?: AttachmentInput[];
}

export async function POST(request: Request) {
    const client = createClient({
        apiKey: process.env.SIMPLE_EMAIL_API_KEY ?? "",
        baseUrl: "http://localhost:8080"
    })

    const body = await request.json() as EmailRequestBody

    const res = await client.send({
        from: body.from,
        to: body.to,
        subject: body.subject,
        body: body.body,
        attachments: body.attachments?.map((att) => {
            if (att.url) {
                return {
                    filename: att.filename,
                    contentType: att.contentType,
                    source: {
                        case: "url" as const,
                        value: att.url
                    }
                }
            }

            if (att.base64) {
                return {
                    filename: att.filename,
                    contentType: att.contentType,
                    source: {
                        case: "base64Content" as const,
                        value: att.base64
                    }
                }
            }

            // Fallback: invalid attachment
            throw new Error("Attachment must have either 'url' or 'base64' property")
        }) ?? []
    })

    return NextResponse.json({ message: "Email sent successfully", result: res }, { status: 200 });
}
