import { apiKeyClient } from '@/lib/connect';
import type {
    ApiKey,
    CreateApiKeyRequest,
    CreateApiKeyResponse,
    UpdateApiKeyRequest,
    ListApiKeysResponse,
} from '@/lib/connect';

/**
 * API Key Service
 * Handles all API key-related operations using Connect RPC
 */
export const apiKeyService = {
    /**
     * Create a new API key
     */
    create: async (data: Omit<CreateApiKeyRequest, '$typeName'>): Promise<CreateApiKeyResponse> => {
        return apiKeyClient().createApiKey(data);
    },

    /**
     * Get an API key by ID
     */
    get: async (id: string): Promise<ApiKey> => {
        return apiKeyClient().getApiKey({ id });
    },

    /**
     * List all API keys for the current user
     */
    list: async (): Promise<ListApiKeysResponse> => {
        return apiKeyClient().listApiKeys({});
    },

    /**
     * Update an API key
     */
    update: async (id: string, data: Omit<UpdateApiKeyRequest, 'id' | '$typeName'>): Promise<ApiKey> => {
        return apiKeyClient().updateApiKey({ id, ...data });
    },

    /**
     * Delete an API key permanently
     */
    delete: async (id: string): Promise<void> => {
        await apiKeyClient().deleteApiKey({ id });
    },

    /**
     * Revoke an API key (soft delete)
     */
    revoke: async (id: string): Promise<ApiKey> => {
        return apiKeyClient().revokeApiKey({ id });
    },
};

export default apiKeyService;
