import { useQuery, keepPreviousData } from '@tanstack/react-query';
import { activityService } from '@/services';
import { queryKeys } from '@/lib/queryClient';
import type { ListActivityLogsRequest } from '@/types';

/**
 * Hook to list activity logs with pagination and filters
 */
export const useActivityLogs = (params: ListActivityLogsRequest = {}) => {
    return useQuery({
        queryKey: queryKeys.activity.list(params),
        queryFn: () => activityService.list(params),
        placeholderData: keepPreviousData, // Keep previous data while fetching new page
        staleTime: 1000 * 30, // 30 seconds
    });
};
