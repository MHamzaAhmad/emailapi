import { createFileRoute } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { Shell, PageHeader } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { Card, CardContent } from '@/components/ui/card'
import { AppPortal } from 'svix-react'
import 'svix-react/style.css'
import { useWebhookPortal } from '@/hooks'

export const Route = createFileRoute('/webhooks')(
    {
        component: WebhooksPage,
    }
)

function WebhooksPage() {
    return (
        <AuthGuard>
            <WebhooksContent />
        </AuthGuard>
    )
}

function WebhooksContent() {
    const { data: portalData, isLoading, error } = useWebhookPortal()
    const [portalUrl, setPortalUrl] = useState<string | null>(null)

    useEffect(() => {
        if (portalData?.url) {
            setPortalUrl(portalData.url)
        }
    }, [portalData])

    return (
        <Shell>
            <PageHeader
                title="Webhooks"
                description="Configure endpoints to receive email events"
            />

            <Card>
                <CardContent className="p-0">
                    {isLoading ? (
                        <div className="p-4 text-center text-xs text-muted-foreground">
                            Loading webhook portal...
                        </div>
                    ) : error ? (
                        <div className="p-4 text-center text-xs text-destructive">
                            Failed to load webhook portal. Please try again.
                        </div>
                    ) : portalUrl ? (
                        <div className="svix-portal-wrapper">
                            <AppPortal url={portalUrl} fullSize />
                        </div>
                    ) : (
                        <div className="p-4 text-center text-xs text-muted-foreground">
                            No webhook portal available.
                        </div>
                    )}
                </CardContent>
            </Card>

            <style>{`
                .svix-portal-wrapper {
                    min-height: 500px;
                }
                .svix-portal-wrapper iframe {
                    border: none;
                    width: 100%;
                    min-height: 500px;
                }
            `}</style>
        </Shell>
    )
}
