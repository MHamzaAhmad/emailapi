/**
 * Connect RPC Client for Email API
 * 
 * This module provides type-safe gRPC-Web clients for all API services.
 * Uses Clerk JWT for authentication.
 */
import { createClient, type Client, type Transport } from '@connectrpc/connect';
import { createGrpcWebTransport } from '@connectrpc/connect-web';

// Import all service descriptors from generated protos
import { DomainService } from '@/generated/v1/domain_pb';
import { EmailService } from '@/generated/v1/email_pb';
import { ApiKeyService } from '@/generated/v1/apikey_pb';
import { UserService } from '@/generated/v1/user_pb';
import { ActivityService } from '@/generated/v1/activity_pb';
import { WebhookService } from '@/generated/v1/webhook_pb';

// API base URL
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

// Token getter - will be set by AuthGuard component
let getToken: (() => Promise<string | null>) | null = null;

/**
 * Set the token getter function (called from AuthGuard with Clerk's getToken)
 */
export const setTokenGetter = (getter: () => Promise<string | null>) => {
    getToken = getter;
};

/**
 * Create a gRPC-Web transport with authentication interceptor
 */
const createAuthenticatedTransport = (): Transport => {
    return createGrpcWebTransport({
        baseUrl: API_BASE_URL,
        interceptors: [
            (next) => async (req) => {
                if (getToken) {
                    try {
                        const token = await getToken();
                        if (token) {
                            req.header.set('Authorization', `Bearer ${token}`);
                        }
                    } catch (error) {
                        console.warn('Failed to get auth token:', error);
                    }
                }
                return next(req);
            },
        ],
    });
};

// Singleton transport instance
let transport: Transport | null = null;

const getTransport = (): Transport => {
    if (!transport) {
        transport = createAuthenticatedTransport();
    }
    return transport;
};

/**
 * Service clients - lazy initialized with authenticated transport
 */
export const domainClient = (): Client<typeof DomainService> =>
    createClient(DomainService, getTransport());

export const emailClient = (): Client<typeof EmailService> =>
    createClient(EmailService, getTransport());

export const apiKeyClient = (): Client<typeof ApiKeyService> =>
    createClient(ApiKeyService, getTransport());

export const userClient = (): Client<typeof UserService> =>
    createClient(UserService, getTransport());

export const activityClient = (): Client<typeof ActivityService> =>
    createClient(ActivityService, getTransport());

export const webhookClient = (): Client<typeof WebhookService> =>
    createClient(WebhookService, getTransport());

// Re-export types for convenience
export type {
    Domain,
    DomainStatus,
    DnsRecord,
    RecordStatus,
    RecordType,
    AddDomainRequest,
    AddDomainResponse,
    GetDomainRequest,
    GetDomainResponse,
    ListDomainsRequest,
    ListDomainsResponse,
    DeleteDomainRequest,
    DeleteDomainResponse,
    VerifyDomainRequest,
    VerifyDomainResponse,
} from '@/generated/v1/domain_pb';

export type {
    SendEmailRequest,
    SendEmailResponse,
    EmailStatus,
    Attachment,
} from '@/generated/v1/email_pb';

export type {
    ApiKey,
    Environment,
    Scope,
    CreateApiKeyRequest,
    CreateApiKeyResponse,
    GetApiKeyRequest,
    ListApiKeysRequest,
    ListApiKeysResponse,
    UpdateApiKeyRequest,
    DeleteApiKeyRequest,
    DeleteApiKeyResponse,
    RevokeApiKeyRequest,
} from '@/generated/v1/apikey_pb';

export type {
    User,
    UserRole,
    CreateUserRequest,
    CreateUserResponse,
    GetCurrentUserRequest,
    UpdateUserRequest,
    ListUsersRequest,
    ListUsersResponse,
} from '@/generated/v1/user_pb';

export type {
    ActivityLog,
    ListActivityLogsRequest,
    ListActivityLogsResponse,
} from '@/generated/v1/activity_pb';

export type {
    GetAppPortalAccessRequest,
    GetAppPortalAccessResponse,
} from '@/generated/v1/webhook_pb';
