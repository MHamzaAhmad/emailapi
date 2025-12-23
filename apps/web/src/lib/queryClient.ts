import { QueryClient } from '@tanstack/react-query';

// Create a query client with default options
export const createQueryClient = () =>
    new QueryClient({
        defaultOptions: {
            queries: {
                // Stale time: how long data is considered fresh (5 minutes)
                staleTime: 1000 * 60 * 5,
                // Cache time: how long inactive data stays in cache (30 minutes)
                gcTime: 1000 * 60 * 30,
                // Retry failed queries 3 times with exponential backoff
                retry: (failureCount, error) => {
                    // Don't retry on client errors (Connect codes that map to 4xx)
                    // Connect codes: 3=InvalidArgument, 5=NotFound, 6=AlreadyExists, 
                    // 7=PermissionDenied, 16=Unauthenticated, 9=FailedPrecondition
                    const connectError = error as { code?: number };
                    const nonRetryableCodes = [3, 5, 6, 7, 9, 16];
                    if (typeof connectError.code === 'number' && nonRetryableCodes.includes(connectError.code)) {
                        return false;
                    }
                    return failureCount < 3;
                },
                // Refetch on window focus for fresh data
                refetchOnWindowFocus: true,
                // Don't refetch on reconnect by default
                refetchOnReconnect: true,
            },
            mutations: {
                // Retry mutations once
                retry: 1,
                // Handle errors globally if needed
                onError: (error) => {
                    console.error('Mutation error:', error);
                },
            },
        },
    });

// Singleton query client for SSR/SSG
let queryClient: QueryClient | undefined;

export const getQueryClient = () => {
    if (typeof window === 'undefined') {
        // Server: always create a new query client
        return createQueryClient();
    }
    // Client: reuse the same query client
    if (!queryClient) {
        queryClient = createQueryClient();
    }
    return queryClient;
};

// Query key factories for type-safe query keys
export const queryKeys = {
    // User keys
    users: {
        all: ['users'] as const,
        list: (params?: any) => [...queryKeys.users.all, 'list', params] as const,
        me: () => [...queryKeys.users.all, 'me'] as const,
        detail: (id: string) => [...queryKeys.users.all, id] as const,
    },

    // API Key keys
    apiKeys: {
        all: ['apiKeys'] as const,
        list: () => [...queryKeys.apiKeys.all, 'list'] as const,
        detail: (id: string) => [...queryKeys.apiKeys.all, id] as const,
    },

    // Domain keys
    domains: {
        all: ['domains'] as const,
        list: () => [...queryKeys.domains.all, 'list'] as const,
        detail: (id: string) => [...queryKeys.domains.all, id] as const,
        records: (id: string) => [...queryKeys.domains.all, id, 'records'] as const,
    },

    // Webhook keys
    webhooks: {
        all: ['webhooks'] as const,
        portal: () => [...queryKeys.webhooks.all, 'portal'] as const,
    },

    // Activity keys
    activity: {
        all: ['activity'] as const,
        list: (params?: any) => [...queryKeys.activity.all, 'list', params] as const,
    },
} as const;

export default getQueryClient;
