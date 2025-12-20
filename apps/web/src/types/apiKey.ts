// API Key types matching the backend API

export type Environment = 'live' | 'dev';

export type Scope =
    | 'email:send'
    | 'email:read'
    | 'domain:read'
    | 'domain:write'
    | 'apikey:read'
    | 'apikey:write'
    | 'user:read'
    | 'user:write';

export interface ApiKey {
    id: string;
    userId: string;
    name: string;
    keyPrefix: string;
    scopes: Scope[];
    environment: Environment;
    isActive: boolean;
    lastUsedAt?: string;
    expiresAt?: string;
    createdAt: string;
    updatedAt: string;
}

// Request DTOs
export interface CreateApiKeyRequest {
    name: string;
    scopes: Scope[];
    environment?: Environment;
    expiresAt?: string;
}

export interface UpdateApiKeyRequest {
    name?: string;
    scopes?: Scope[];
    isActive?: boolean;
    expiresAt?: string;
}

// Response DTOs
export interface CreateApiKeyResponse {
    apiKey: ApiKey;
    rawKey: string;
    message: string;
}

export interface ListApiKeysResponse {
    data: ApiKey[];
}
