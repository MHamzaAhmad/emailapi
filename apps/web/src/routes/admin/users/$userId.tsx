import { createFileRoute, Link } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    ArrowLeft01Icon,
    StopCircleIcon,
    PlayCircleIcon,
    Alert02Icon,
    CheckmarkCircle02Icon,
    Mail01Icon,
    User02Icon,
} from '@hugeicons/core-free-icons'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useUserDetails, useSuspendUser, useUnsuspendUser } from '@/hooks'
import { formatDate } from '@/lib/utils'
import { useState } from 'react'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

export const Route = createFileRoute('/admin/users/$userId')({
    component: UserDetailsPage,
})

function UserDetailsPage() {
    const { userId } = Route.useParams()
    const { data: user, isLoading, isError, error } = useUserDetails(userId)
    const suspendMutation = useSuspendUser()
    const unsuspendMutation = useUnsuspendUser()

    const [showSuspendDialog, setShowSuspendDialog] = useState(false)
    const [suspendReason, setSuspendReason] = useState('')

    if (isLoading) {
        return (
            <div className="animate-pulse text-muted-foreground text-sm py-8 text-center">
                Loading user details...
            </div>
        )
    }

    if (isError || !user) {
        return (
            <div className="text-destructive text-sm py-8 text-center">
                Failed to load user. {error?.message}
            </div>
        )
    }

    const handleSuspend = async () => {
        if (!suspendReason.trim()) return
        await suspendMutation.mutateAsync({ userId, reason: suspendReason })
        setShowSuspendDialog(false)
        setSuspendReason('')
    }

    const handleUnsuspend = async () => {
        await unsuspendMutation.mutateAsync(userId)
    }

    return (
        <>
            {/* Back Link */}
            <Link
                to="/admin/users"
                className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground mb-6"
            >
                <HugeiconsIcon icon={ArrowLeft01Icon} size={16} />
                Back to Users
            </Link>

            {/* User Header */}
            <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-4 mb-8">
                <div className="space-y-1">
                    <h1 className="text-2xl font-bold tracking-tight">{user.name}</h1>
                    <div className="flex items-center gap-2 text-muted-foreground">
                        <HugeiconsIcon icon={Mail01Icon} size={14} />
                        <span className="text-sm">{user.email}</span>
                    </div>
                    <div className="flex items-center gap-2 text-muted-foreground">
                        <HugeiconsIcon icon={User02Icon} size={14} />
                        <Badge variant={user.role === 'admin' ? 'default' : 'outline'} className="text-xs">
                            {user.role}
                        </Badge>
                        {user.isSuspended && (
                            <Badge variant="destructive" className="text-xs">Suspended</Badge>
                        )}
                        {!user.isSuspended && user.isFlagged && (
                            <Badge variant="warning" className="text-xs">Flagged</Badge>
                        )}
                    </div>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-2">
                    {user.isSuspended ? (
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={handleUnsuspend}
                            disabled={unsuspendMutation.isPending}
                            className="gap-2"
                        >
                            <HugeiconsIcon icon={PlayCircleIcon} size={14} />
                            {unsuspendMutation.isPending ? 'Unsuspending...' : 'Unsuspend'}
                        </Button>
                    ) : (
                        <Button
                            variant="destructive"
                            size="sm"
                            onClick={() => setShowSuspendDialog(true)}
                            className="gap-2"
                        >
                            <HugeiconsIcon icon={StopCircleIcon} size={14} />
                            Suspend
                        </Button>
                    )}
                </div>
            </div>

            {/* Suspension Info */}
            {user.isSuspended && (
                <div className="mb-6 p-4 rounded-lg border border-destructive/20 bg-destructive/5">
                    <div className="flex items-start gap-3">
                        <HugeiconsIcon icon={Alert02Icon} size={18} className="text-destructive mt-0.5" />
                        <div className="space-y-1">
                            <div className="font-medium text-destructive text-sm">Account Suspended</div>
                            {user.suspensionReason && (
                                <div className="text-sm text-muted-foreground">
                                    Reason: {user.suspensionReason}
                                </div>
                            )}
                            {user.suspendedAt && (
                                <div className="text-xs text-muted-foreground">
                                    Since {formatDate(new Date(Number(user.suspendedAt.seconds) * 1000))}
                                    {user.suspendedBy && ` by ${user.suspendedBy}`}
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* Stats Grid */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
                <StatCard
                    title="Total Bounces"
                    value={user.totalBounces}
                    variant={user.totalBounces > 10 ? 'warning' : 'default'}
                />
                <StatCard
                    title="Hard Bounces"
                    value={user.hardBounces}
                    variant={user.hardBounces > 5 ? 'destructive' : 'default'}
                />
                <StatCard
                    title="Complaints"
                    value={user.complaints}
                    variant={user.complaints > 0 ? 'destructive' : 'default'}
                />
                <StatCard
                    title="Suspension Score"
                    value={user.suspensionScore.toFixed(1)}
                    variant={user.suspensionScore > 50 ? 'destructive' : user.suspensionScore > 25 ? 'warning' : 'default'}
                />
            </div>

            {/* 30-Day Stats */}
            <div className="rounded-xl border border-border/40 bg-card shadow-sm p-6 mb-8">
                <h3 className="font-semibold mb-4 text-sm">Last 30 Days</h3>
                <div className="grid grid-cols-2 gap-4">
                    <div className="flex items-center justify-between">
                        <span className="text-sm text-muted-foreground">Bounces</span>
                        <span className="font-medium">{user.bounces30d}</span>
                    </div>
                    <div className="flex items-center justify-between">
                        <span className="text-sm text-muted-foreground">Complaints</span>
                        <span className="font-medium">{user.complaints30d}</span>
                    </div>
                </div>
            </div>

            {/* User Info */}
            <div className="rounded-xl border border-border/40 bg-card shadow-sm p-6">
                <h3 className="font-semibold mb-4 text-sm">Account Info</h3>
                <div className="space-y-3 text-sm">
                    <div className="flex items-center justify-between">
                        <span className="text-muted-foreground">User ID</span>
                        <code className="text-xs bg-muted px-2 py-1 rounded">{user.id}</code>
                    </div>
                    <div className="flex items-center justify-between">
                        <span className="text-muted-foreground">Created</span>
                        <span>
                            {user.createdAt
                                ? formatDate(new Date(Number(user.createdAt.seconds) * 1000))
                                : '-'}
                        </span>
                    </div>
                    <div className="flex items-center justify-between">
                        <span className="text-muted-foreground">Active</span>
                        <HugeiconsIcon
                            icon={user.isActive ? CheckmarkCircle02Icon : StopCircleIcon}
                            size={16}
                            className={user.isActive ? 'text-green-500' : 'text-muted-foreground'}
                        />
                    </div>
                </div>
            </div>

            {/* Suspend Dialog */}
            <Dialog open={showSuspendDialog} onOpenChange={setShowSuspendDialog}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Suspend User</DialogTitle>
                        <DialogDescription>
                            This will prevent the user from sending emails. They will see a banner in their dashboard.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="reason">Suspension Reason</Label>
                            <Textarea
                                id="reason"
                                placeholder="Enter the reason for suspension..."
                                value={suspendReason}
                                onChange={(e) => setSuspendReason(e.target.value)}
                                rows={3}
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setShowSuspendDialog(false)}>
                            Cancel
                        </Button>
                        <Button
                            variant="destructive"
                            onClick={handleSuspend}
                            disabled={!suspendReason.trim() || suspendMutation.isPending}
                        >
                            {suspendMutation.isPending ? 'Suspending...' : 'Suspend User'}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </>
    )
}

interface StatCardProps {
    title: string
    value: string | number
    variant?: 'default' | 'warning' | 'destructive'
}

function StatCard({ title, value, variant = 'default' }: StatCardProps) {
    const colorClass = {
        default: 'text-foreground',
        warning: 'text-amber-600 dark:text-amber-400',
        destructive: 'text-destructive',
    }[variant]

    return (
        <div className="rounded-lg border border-border/40 bg-card p-4">
            <div className="text-xs text-muted-foreground mb-1">{title}</div>
            <div className={`text-2xl font-bold ${colorClass}`}>{value}</div>
        </div>
    )
}
