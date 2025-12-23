import { createFileRoute } from '@tanstack/react-router'
import { useUser } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    MailIcon,
    CheckmarkCircleIcon,
    AlertCircleIcon,
    ClockIcon,
} from '@hugeicons/core-free-icons'
import { Shell, PageHeader } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

export const Route = createFileRoute('/dashboard')(
    {
        component: DashboardPage,
    }
)

function DashboardPage() {
    return (
        <AuthGuard>
            <DashboardContent />
        </AuthGuard>
    )
}

function DashboardContent() {
    const { user } = useUser()

    // Mock stats - these will come from API
    const stats = {
        totalSent: 1234,
        delivered: 1180,
        bounced: 12,
        pending: 42,
    }

    // Mock activity - will come from API
    const activity = [
        { id: '1', type: 'sent', message: 'Email sent to user@example.com', time: '2 min ago' },
        { id: '2', type: 'delivered', message: 'Email delivered to hello@company.com', time: '5 min ago' },
        { id: '3', type: 'bounced', message: 'Bounce: invalid@nowhere.com', time: '12 min ago' },
        { id: '4', type: 'sent', message: 'Email sent to team@startup.io', time: '15 min ago' },
        { id: '5', type: 'delivered', message: 'Email delivered to dev@example.com', time: '20 min ago' },
    ]

    return (
        <Shell>
            <PageHeader
                title="Dashboard"
                description={`Welcome back${user?.firstName ? `, ${user.firstName}` : ''}`}
            />

            {/* Stats */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
                <StatCard
                    label="Sent"
                    value={stats.totalSent}
                    icon={MailIcon}
                />
                <StatCard
                    label="Delivered"
                    value={stats.delivered}
                    icon={CheckmarkCircleIcon}
                />
                <StatCard
                    label="Bounced"
                    value={stats.bounced}
                    icon={AlertCircleIcon}
                    variant="destructive"
                />
                <StatCard
                    label="Pending"
                    value={stats.pending}
                    icon={ClockIcon}
                />
            </div>

            {/* Activity Log */}
            <Card>
                <CardHeader className="pb-2">
                    <CardTitle>Activity</CardTitle>
                </CardHeader>
                <CardContent className="p-0">
                    <div className="divide-y divide-border">
                        {activity.map((item) => (
                            <div key={item.id} className="flex items-center justify-between px-3 py-2">
                                <div className="flex items-center gap-2">
                                    <ActivityIcon type={item.type} />
                                    <span className="text-xs">{item.message}</span>
                                </div>
                                <span className="text-2xs text-muted-foreground">{item.time}</span>
                            </div>
                        ))}
                    </div>
                </CardContent>
            </Card>
        </Shell>
    )
}

function StatCard({
    label,
    value,
    icon,
    variant,
}: {
    label: string
    value: number
    icon: typeof MailIcon
    variant?: 'destructive'
}) {
    return (
        <div className="stat-card">
            <div className="flex items-center justify-between mb-1">
                <span className="text-2xs text-muted-foreground">{label}</span>
                <span className={variant === 'destructive' ? 'text-destructive' : 'text-muted-foreground'}>
                    <HugeiconsIcon icon={icon} size={14} strokeWidth={1.5} />
                </span>
            </div>
            <div className={`text-xl font-medium ${variant === 'destructive' ? 'text-destructive' : ''}`}>
                {value.toLocaleString()}
            </div>
        </div>
    )
}

function ActivityIcon({ type }: { type: string }) {
    switch (type) {
        case 'sent':
            return <Badge variant="secondary">sent</Badge>
        case 'delivered':
            return <Badge variant="success">delivered</Badge>
        case 'bounced':
            return <Badge variant="destructive">bounced</Badge>
        default:
            return <Badge variant="outline">{type}</Badge>
    }
}
