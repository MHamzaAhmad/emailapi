import api from '@/lib/api';
import type {
    ApiKey,
    CreateApiKeyRequest,
    CreateApiKeyResponse,
    UpdateApiKeyRequest,
    ListApiKeysResponse,
} from '@/types';

/**
 * API Key Service
 * Handles all API key-related operations
 */
export const apiKeyService = {
    /**
     * Create a new API key
     */
    create: async (data: CreateApiKeyRequest): Promise<CreateApiKeyResponse> => {
        return api.post<CreateApiKeyResponse>('/v1/api-keys', data);
    },

    /**
     * Get an API key by ID
     */
    get: async (id: string): Promise<ApiKey> => {
        return api.get<ApiKey>(`/v1/api-keys/${id}`);
    },

    /**
     * List all API keys for the current user
     */
    list: async (): Promise<ListApiKeysResponse> => {
        return api.get<ListApiKeysResponse>('/v1/api-keys');
    },

    /**
     * Update an API key
     */
    update: async (id: string, data: UpdateApiKeyRequest): Promise<ApiKey> => {
        return api.patch<ApiKey>(`/v1/api-keys/${id}`, data);
    },

    /**
     * Delete an API key permanently
     */
    delete: async (id: string): Promise<void> => {
        return api.delete(`/v1/api-keys/${id}`);
    },

    /**
     * Revoke an API key (soft delete)
     */
    revoke: async (id: string): Promise<ApiKey> => {
        return api.post<ApiKey>(`/v1/api-keys/${id}/revoke`);
    },
};

export default apiKeyService;
