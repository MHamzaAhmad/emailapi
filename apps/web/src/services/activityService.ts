import { activityClient } from '@/lib/connect';
import type { ListActivityLogsRequest, ListActivityLogsResponse } from '@/lib/connect';

/**
 * Activity Service
 * Handles activity log related API operations using Connect RPC
 */
export const activityService = {
    /**
     * List activity logs with pagination and filters
     */
    list: async (params: Partial<Omit<ListActivityLogsRequest, '$typeName'>>): Promise<ListActivityLogsResponse> => {
        return activityClient().listActivityLogs({
            pageSize: params.pageSize ?? 20,
            offset: params.offset ?? 0,
            entityType: params.entityType,
            action: params.action,
            startTime: params.startTime,
            endTime: params.endTime,
        });
    },
};

export default activityService;
