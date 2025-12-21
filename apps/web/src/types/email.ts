export enum EmailStatus {
    EMAIL_STATUS_UNSPECIFIED = 'EMAIL_STATUS_UNSPECIFIED',
    EMAIL_STATUS_PENDING = 'EMAIL_STATUS_PENDING',
    EMAIL_STATUS_PROCESSING_ATTACHMENTS = 'EMAIL_STATUS_PROCESSING_ATTACHMENTS',
    EMAIL_STATUS_SCANNING_ATTACHMENTS = 'EMAIL_STATUS_SCANNING_ATTACHMENTS',
    EMAIL_STATUS_SCAN_FAILED = 'EMAIL_STATUS_SCAN_FAILED',
    EMAIL_STATUS_QUEUED = 'EMAIL_STATUS_QUEUED',
    EMAIL_STATUS_SENT = 'EMAIL_STATUS_SENT',
    EMAIL_STATUS_DELIVERED = 'EMAIL_STATUS_DELIVERED',
    EMAIL_STATUS_FAILED = 'EMAIL_STATUS_FAILED',
    EMAIL_STATUS_BOUNCED = 'EMAIL_STATUS_BOUNCED',
}

export enum EmailCategory {
    EMAIL_CATEGORY_UNSPECIFIED = 'EMAIL_CATEGORY_UNSPECIFIED',
    EMAIL_CATEGORY_ACTIVE = 'EMAIL_CATEGORY_ACTIVE',
    EMAIL_CATEGORY_ARCHIVED = 'EMAIL_CATEGORY_ARCHIVED',
}

export interface Attachment {
    filename: string;
    contentType: string;
    url?: string;
    base64Content?: string;
}

export interface EmailAttachment {
    id: string;
    filename: string;
    contentType: string;
    sizeBytes: number;
    scanStatus: string;
}

export interface SendEmailRequest {
    from: string;
    to: string[];
    cc?: string[];
    bcc?: string[];
    subject: string;
    body?: string;
    html?: string;
    metadata?: Record<string, string>;
    scheduledAt?: string; // ISO string
    attachments?: Attachment[];
}

export interface SendEmailResponse {
    id: string;
    status: EmailStatus;
}

export interface Email {
    id: string;
    from: string;
    to: string[];
    cc: string[];
    bcc: string[];
    subject: string;
    body: string;
    html: string;
    status: EmailStatus;
    providerId: string;
    userId: string;
    metadata: Record<string, string>;
    scheduledAt?: string;
    sentAt?: string;
    createdAt: string;
    updatedAt: string;
    attachments: EmailAttachment[];
    errorMessage: string;
}

export interface ListEmailsRequest {
    category?: EmailCategory;
    limit?: number;
    offset?: number;
}

export interface ListEmailsResponse {
    data: Email[];
    limit: number;
    offset: number;
    totalCount: number;
    category: EmailCategory;
}
