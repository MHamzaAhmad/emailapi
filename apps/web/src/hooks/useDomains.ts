import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { domainService } from '@/services';
import { queryKeys } from '@/lib/queryClient';
import type {
    Domain,
    AddDomainRequest,
    SetMailFromRequest,
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
 * Hook to get DNS records for a domain
 */
export const useDomainRecords = (id: string) => {
    return useQuery({
        queryKey: queryKeys.domains.records(id),
        queryFn: () => domainService.getRecords(id),
        enabled: !!id,
    });
};

/**
 * Hook to add a new domain
 */
export const useAddDomain = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: AddDomainRequest) => domainService.add(data),
        onSuccess: (response) => {
            // Invalidate the list to show the new domain
            queryClient.invalidateQueries({ queryKey: queryKeys.domains.list() });
            // Pre-populate the domain detail cache
            queryClient.setQueryData(
                queryKeys.domains.detail(response.domain.id),
                response.domain
            );
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
            queryClient.removeQueries({ queryKey: queryKeys.domains.records(id) });
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
            // Also invalidate records as they might have updated statuses
            queryClient.invalidateQueries({ queryKey: queryKeys.domains.records(id) });
        },
    });
};

/**
 * Hook to set custom MAIL FROM domain
 */
export const useSetMailFrom = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: SetMailFromRequest }) =>
            domainService.setMailFrom(id, data),
        onSuccess: (updatedDomain, variables) => {
            // Update the domain cache
            queryClient.setQueryData<Domain>(
                queryKeys.domains.detail(variables.id),
                updatedDomain
            );
            // Invalidate records as new MAIL FROM records will be needed
            queryClient.invalidateQueries({
                queryKey: queryKeys.domains.records(variables.id),
            });
            // Invalidate the list
            queryClient.invalidateQueries({ queryKey: queryKeys.domains.list() });
        },
    });
};
