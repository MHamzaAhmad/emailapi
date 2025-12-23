import { createFileRoute } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    WebhookIcon,
    Add01Icon,
    RefreshIcon,
    Alert01Icon,
} from '@hugeicons/core-free-icons'
import { Shell, PageHeader } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { useState } from 'react'

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
    const [endpoints] = useState([
        { id: '1', url: 'https://api.myapp.com/webhooks/email', events: ['email.sent', 'email.delivered'], status: 'active' },
    ])

    return (
        <Shell>
            <PageHeader
                title="Webhook Portal"
                description="Broadcast events to your external services in realtime."
                actions={
                    <button className="btn-tiny flex items-center gap-2">
                        <HugeiconsIcon icon={Add01Icon} size={10} strokeWidth={2.5} />
                        <span>Add Endpoint</span>
                    </button>
                }
            />

            <div className="space-y-6">
                {endpoints.length > 0 ? (
                    <div className="border border-foreground bg-card">
                        <div className="px-4 py-3 border-b border-foreground bg-muted/20 flex items-center justify-between">
                            <h2 className="text-[10px] font-bold uppercase tracking-widest">Configured Endpoints</h2>
                            <button className="text-[9px] font-bold uppercase tracking-widest hover:text-foreground/60 transition-colors flex items-center gap-1">
                                <HugeiconsIcon icon={RefreshIcon} size={8} strokeWidth={3} />
                                <span>Refresh</span>
                            </button>
                        </div>
                        <div className="divide-y divide-foreground/10">
                            {endpoints.map((ep) => (
                                <div key={ep.id} className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                                    <div className="space-y-1">
                                        <div className="flex items-center gap-2">
                                            <span className="text-xs font-bold tracking-tight">{ep.url}</span>
                                            <span className="text-[9px] font-bold uppercase tracking-widest px-1.5 py-0.5 bg-foreground text-background">
                                                {ep.status}
                                            </span>
                                        </div>
                                        <div className="flex flex-wrap gap-1">
                                            {ep.events.map(event => (
                                                <span key={event} className="text-[9px] font-mono text-muted-foreground border border-foreground/10 px-1">
                                                    {event}
                                                </span>
                                            ))}
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <button className="btn-tiny">
                                            View Logs
                                        </button>
                                        <button className="btn-tiny border-destructive/20 text-destructive hover:bg-destructive hover:text-background">
                                            Delete
                                        </button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                ) : (
                    <div className="border border-sharp-bold border-dashed p-12 text-center bg-muted/5">
                        <HugeiconsIcon icon={WebhookIcon} size={32} strokeWidth={1} className="mx-auto mb-4 text-muted-foreground/30" />
                        <h3 className="text-xs font-bold uppercase tracking-widest mb-1 text-muted-foreground">No endpoints found</h3>
                        <p className="text-[11px] text-muted-foreground mb-6">Start listening to events by adding your first endpoint.</p>
                        <button className="btn-tiny mx-auto">Create Endpoint</button>
                    </div>
                )}

                <div className="p-6 border border-foreground/10 bg-muted/5">
                    <div className="flex items-start gap-3">
                        <HugeiconsIcon icon={Alert01Icon} size={14} strokeWidth={2} className="text-foreground shrink-0 mt-1" />
                        <div>
                            <h4 className="text-[10px] font-bold uppercase tracking-wider mb-1">Webhook Security</h4>
                            <p className="text-[11px] text-muted-foreground leading-relaxed">
                                All webhook events are signed with a unique secret per endpoint.
                                We recommend verifying the signature using our official SDKs to ensure data integrity.
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        </Shell>
    )
}
