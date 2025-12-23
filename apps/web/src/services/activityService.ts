import api from '@/lib/api';
import type { ListActivityLogsRequest, ListActivityLogsResponse } from '@/types';

/**
 * Activity Service
 * Handles activity log related API operations
 */
export const activityService = {
    /**
     * List activity logs with pagination and filters
     */
    list: async (params: ListActivityLogsRequest): Promise<ListActivityLogsResponse> => {
        return api.get<ListActivityLogsResponse>('/v1/activity-logs', { params });
    },
};

export default activityService;
