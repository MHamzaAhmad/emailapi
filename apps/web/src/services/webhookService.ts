import { webhookClient } from '@/lib/connect';
import type { GetAppPortalAccessResponse } from '@/lib/connect';

/**
 * Webhook Service
 * Handles webhook-related API operations using Connect RPC
 */
export const webhookService = {
    /**
     * Get App Portal access URL and token
     */
    getAppPortalAccess: async (): Promise<GetAppPortalAccessResponse> => {
        return webhookClient().getAppPortalAccess({});
    },
};

export default webhookService;
