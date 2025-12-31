import { useQuery } from '@tanstack/react-query';
import { userClient } from '@/lib/connect';
import { queryKeys } from '@/lib/queryClient';
import type { SuspensionStatus } from '@/lib/connect';

/**
 * Hook to get the current user's suspension status for FE banner
 */
export const useSuspensionStatus = () => {
    return useQuery<SuspensionStatus>({
        queryKey: queryKeys.suspensionStatus,
        queryFn: () => userClient().getSuspensionStatus({}),
        // Cache for 5 minutes but refetch on focus
        staleTime: 1000 * 60 * 5,
        refetchOnWindowFocus: true,
    });
};

export default useSuspensionStatus;
