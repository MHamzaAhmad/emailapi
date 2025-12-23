import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { domainService } from '@/services';
import { queryKeys } from '@/lib/queryClient';
import type {
    Domain,
    AddDomainRequest,
    ListDomainsResponse,
} from '@/types';

/**
 * Hook to list all domains
 */
export const useDomains = () => {
    return useQuery({
        queryKey: queryKeys.domains.list(),
        queryFn: domainService.list,
        select: (data: ListDomainsResponse) => data.data,
    });
};

/**
 * Hook to get a single domain by ID
 */
export const useDomain = (id: string) => {
    return useQuery({
        queryKey: queryKeys.domains.detail(id),
        queryFn: () => domainService.get(id),
        enabled: !!id,
    });
};

/**
 * Hook to add a new domain
 */
export const useAddDomain = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: Omit<AddDomainRequest, '$typeName'>) => domainService.add(data),
        onSuccess: (response) => {
            if (response.domain) {
                // Invalidate the list to show the new domain
                queryClient.invalidateQueries({ queryKey: queryKeys.domains.list() });
                // Pre-populate the domain detail cache
                queryClient.setQueryData(
                    queryKeys.domains.detail(response.domain.id),
                    response.domain
                );
            }
        },
    });
};

/**
 * Hook to delete a domain
 */
export const useDeleteDomain = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => domainService.delete(id),
        onSuccess: (_, id) => {
            // Remove from cache
            queryClient.removeQueries({ queryKey: queryKeys.domains.detail(id) });
            // Invalidate the list
            queryClient.invalidateQueries({ queryKey: queryKeys.domains.list() });
        },
    });
};

/**
 * Hook to verify a domain's DNS configuration
 */
export const useVerifyDomain = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => domainService.verify(id),
        onSuccess: (response, id) => {
            // Update the domain in cache with new verification status
            queryClient.setQueryData<Domain>(
                queryKeys.domains.detail(id),
                response.domain
            );
            // Invalidate the list to reflect status changes
            queryClient.invalidateQueries({ queryKey: queryKeys.domains.list() });
        },
    });
};
