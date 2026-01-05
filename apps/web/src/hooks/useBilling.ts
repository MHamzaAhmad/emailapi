import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { billingClient } from '@/lib/connect';
import { queryKeys } from '@/lib/queryClient';

/**
 * Hook to get available pricing plans
 */
export const usePlans = () => {
    return useQuery({
        queryKey: ['billing', 'plans'],
        queryFn: async () => {
            return billingClient().getPlans({});
        },
        staleTime: 1000 * 60 * 60, // Plans rarely change, cache for 1 hour
    });
};

/**
 * Hook to get current user's subscription info
 */
export const useCurrentSubscription = () => {
    return useQuery({
        queryKey: ['billing', 'subscription'],
        queryFn: async () => {
            return billingClient().syncSubscription({});
        },
    });
};

/**
 * Hook to create a checkout session for plan upgrade
 */
export const useCreateCheckoutSession = () => {
    return useMutation({
        mutationFn: async ({ planId, successUrl }: { planId: string; successUrl: string }) => {
            return billingClient().createCheckoutSession({ planId, successUrl });
        },
    });
};

/**
 * Hook to get customer portal URL
 */
export const useGetCustomerPortalUrl = () => {
    return useMutation({
        mutationFn: async () => {
            return billingClient().getCustomerPortalUrl({});
        },
    });
};

/**
 * Hook to sync subscription after checkout redirect
 */
export const useSyncSubscription = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async () => {
            return billingClient().syncSubscription({});
        },
        onSuccess: () => {
            // Invalidate subscription and user queries after sync
            queryClient.invalidateQueries({ queryKey: ['billing', 'subscription'] });
            queryClient.invalidateQueries({ queryKey: queryKeys.users.me() });
        },
    });
};

// Re-export Plan type for convenience
export type { Plan } from '@/generated/v1/billing_pb';
