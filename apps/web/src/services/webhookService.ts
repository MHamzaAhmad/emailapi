import api from '@/lib/api';
import { AppPortalAccessResponse } from '@/types/webhook';

export const webhookService = {
    /**
     * Get App Portal access URL and token
     */
    getAppPortalAccess: async (): Promise<AppPortalAccessResponse> => {
        return api.post<AppPortalAccessResponse>('/v1/webhooks/portal', {});
    },
};

export default webhookService;
