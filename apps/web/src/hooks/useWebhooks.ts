import { useQuery } from '@tanstack/react-query'
import { webhookService } from '@/services'
import { queryKeys } from '@/lib/queryClient'

/**
 * Hook to get webhook portal access URL
 */
export const useWebhookPortal = () => {
    return useQuery({
        queryKey: queryKeys.webhooks.portal(),
        queryFn: webhookService.getAppPortalAccess,
        staleTime: 1000 * 60 * 5, // 5 minutes
    })
}
