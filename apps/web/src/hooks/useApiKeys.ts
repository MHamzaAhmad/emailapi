import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiKeyService } from '@/services';
import { queryKeys } from '@/lib/queryClient';
import type {
    ApiKey,
    CreateApiKeyRequest,
    UpdateApiKeyRequest,
    ListApiKeysResponse,
} from '@/types';

/**
 * Hook to list all API keys
 */
export const useApiKeys = () => {
    return useQuery({
        queryKey: queryKeys.apiKeys.list(),
        queryFn: apiKeyService.list,
        select: (data: ListApiKeysResponse) => data.data,
    });
};

/**
 * Hook to get a single API key by ID
 */
export const useApiKey = (id: string) => {
    return useQuery({
        queryKey: queryKeys.apiKeys.detail(id),
        queryFn: () => apiKeyService.get(id),
        enabled: !!id,
    });
};

/**
 * Hook to create a new API key
 * Returns the raw key which should be shown to user immediately
 */
export const useCreateApiKey = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateApiKeyRequest) => apiKeyService.create(data),
        onSuccess: () => {
            // Invalidate the list to show the new key
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.list() });
        },
    });
};

/**
 * Hook to update an API key
 */
export const useUpdateApiKey = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateApiKeyRequest }) =>
            apiKeyService.update(id, data),
        onSuccess: (updatedKey, variables) => {
            // Update the specific key in cache
            queryClient.setQueryData<ApiKey>(
                queryKeys.apiKeys.detail(variables.id),
                updatedKey
            );
            // Invalidate the list
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.list() });
        },
    });
};

/**
 * Hook to delete an API key permanently
 */
export const useDeleteApiKey = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => apiKeyService.delete(id),
        onSuccess: (_, id) => {
            // Remove from cache
            queryClient.removeQueries({ queryKey: queryKeys.apiKeys.detail(id) });
            // Invalidate the list
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.list() });
        },
    });
};

/**
 * Hook to revoke an API key (soft delete)
 */
export const useRevokeApiKey = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => apiKeyService.revoke(id),
        onSuccess: (revokedKey, id) => {
            // Update the cache with revoked state
            queryClient.setQueryData<ApiKey>(queryKeys.apiKeys.detail(id), revokedKey);
            // Invalidate the list to reflect the change
            queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.list() });
        },
    });
};
