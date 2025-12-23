import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import { apiKeyClient } from '@/lib/connect';
import { queryKeys } from '@/lib/queryClient';
import type {
    ApiKey,
    CreateApiKeyRequest,
    UpdateApiKeyRequest
} from '@/generated/v1/apikey_pb';
import { ListApiKeysResponse } from '@/generated/v1/apikey_pb';

/**
 * Hook to list all API keys
 */
export const useAPIKeys = (page = 1, pageSize = 10) => {
    return useQuery({
        queryKey: [...queryKeys.apiKeys.list(), page, pageSize],
        queryFn: async () => {
            return apiKeyClient().listApiKeys({
                page,
                pageSize,
            });
        },
    });
};

/**
 * Hook to get a single API key by ID
 */
export const useAPIKey = (id: string) => {
    return useQuery({
        queryKey: queryKeys.apiKeys.detail(id),
        queryFn: async () => {
            return apiKeyClient().getApiKey({ id });
        },
        enabled: !!id,
    });
};

/**
 * Hook to create a new API key
 */
export const useCreateAPIKey = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: Omit<CreateApiKeyRequest, '$typeName'>) => {
            return apiKeyClient().createApiKey(data);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.list() });
        },
    });
};

/**
 * Hook to update an API key
 */
export const useUpdateAPIKey = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: Omit<UpdateApiKeyRequest, '$typeName'>) => {
            return apiKeyClient().updateApiKey(data);
        },
        onSuccess: (response) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.list() });
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.detail(response.id) });
        },
    });
};

/**
 * Hook to delete an API key
 */
export const useDeleteAPIKey = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (id: string) => {
            return apiKeyClient().deleteApiKey({ id });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.list() });
        },
    });
};

/**
 * Hook to revoke an API key
 */
export const useRevokeAPIKey = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (id: string) => {
            return apiKeyClient().revokeApiKey({ id });
        },
        onSuccess: (response) => {
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.list() });
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.detail(response.id) });
        },
    });
};
