import { useQuery, useMutation } from '@tanstack/react-query';
import { billingClient } from '@/lib/connect';

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
            return billingClient().getSubscription({});
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



// Re-export Plan type for convenience
export type { Plan } from '@/generated/v1/billing_pb';
