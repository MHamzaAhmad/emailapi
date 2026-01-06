import { createFileRoute } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Alert02Icon,
    ViewIcon,
} from '@hugeicons/core-free-icons'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { PageHeader } from '@/components/shell'
import { ColumnDef } from '@tanstack/react-table'
import { useFlaggedUsers } from '@/hooks'
import type { FlaggedUser } from '@/lib/connect'
import { formatDate } from '@/lib/utils'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'

export const Route = createFileRoute('/admin/flagged')({
    component: FlaggedUsersPage,
})

const columns: ColumnDef<FlaggedUser>[] = [
    {
        accessorKey: 'email',
        header: 'Email',
        cell: ({ row }) => (
            <div className="flex flex-col">
                <span className="font-medium text-sm">{row.original.email}</span>
                <span className="text-xs text-muted-foreground">{row.original.name}</span>
            </div>
        ),
    },
    {
        accessorKey: 'suspensionScore',
        header: 'Score',
        cell: ({ row }) => {
            const score = row.original.suspensionScore
            const variant = score > 50 ? 'destructive' : score > 25 ? 'warning' : 'outline'
            return (
                <Badge variant={variant} className="text-xs font-mono">
                    {score.toFixed(1)}
                </Badge>
            )
        },
    },
    {
        accessorKey: 'totalBounces',
        header: 'Bounces',
        cell: ({ row }) => (
            <span className="text-sm font-mono">
                {row.original.totalBounces}
                <span className="text-muted-foreground text-xs ml-1">
                    ({row.original.hardBounces} hard)
                </span>
            </span>
        ),
    },
    {
        accessorKey: 'complaints',
        header: 'Complaints',
        cell: ({ row }) => (
            <span className={`text-sm font-mono ${row.original.complaints > 0 ? 'text-destructive' : ''}`}>
                {row.original.complaints}
            </span>
        ),
    },
    {
        accessorKey: 'flaggedReason',
        header: 'Reason',
        cell: ({ row }) => (
            <span className="text-xs text-muted-foreground max-w-[200px] truncate block">
                {row.original.flaggedReason || 'Auto-flagged'}
            </span>
        ),
    },
    {
        accessorKey: 'flaggedAt',
        header: 'Flagged',
        cell: ({ row }) => {
            const ts = row.original.flaggedAt
            if (!ts) return <span className="text-xs text-muted-foreground">-</span>
            const date = new Date(Number(ts.seconds) * 1000)
            return <span className="text-xs text-muted-foreground">{formatDate(date)}</span>
        },
    },
    {
        id: 'actions',
        header: '',
        cell: ({ row }) => (
            <div className="flex items-center gap-1 justify-end">
                <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
                    <Link to="/admin/users/$userId" params={{ userId: row.original.userId }}>
                        <HugeiconsIcon icon={ViewIcon} size={14} />
                    </Link>
                </Button>
            </div>
        ),
    },
]

function FlaggedUsersPage() {
    const [pagination, setPagination] = useState({
        pageIndex: 0,
        pageSize: 50,
    })

    const { data, isLoading, isError, error } = useFlaggedUsers({
        pageSize: pagination.pageSize,
        offset: pagination.pageIndex * pagination.pageSize,
    })

    const totalCount = data?.totalCount || 0
    const pageCount = Math.ceil(totalCount / pagination.pageSize)

    return (
        <>
            <PageHeader
                title="Flagged Users"
                description="Users flagged for review due to reputation issues"
            />

            {/* Info Banner */}
            <div className="mb-6 p-4 rounded-lg border border-amber-500/20 bg-amber-500/5">
                <div className="flex items-start gap-3">
                    <HugeiconsIcon icon={Alert02Icon} size={18} className="text-amber-600 dark:text-amber-400 mt-0.5" />
                    <div className="text-sm text-amber-700 dark:text-amber-300">
                        These users have been automatically flagged based on their email reputation metrics.
                        Review their activity and consider suspension if necessary.
                    </div>
                </div>
            </div>

            {/* Table */}
            <div className="rounded-xl border border-border/40 bg-card shadow-sm overflow-hidden">
                {isError ? (
                    <div className="p-8 text-center text-sm text-destructive">
                        Failed to load flagged users. {error?.message}
                    </div>
                ) : (
                    <DataTable
                        columns={columns}
                        data={data?.users ?? []}
                        isLoading={isLoading}
                        manualPagination={true}
                        pageCount={pageCount}
                        pageIndex={pagination.pageIndex}
                        pageSize={pagination.pageSize}
                        onPaginationChange={setPagination}
                    />
                )}
            </div>

            <div className="mt-4 text-xs text-muted-foreground/60 text-right">
                {totalCount} flagged users
            </div>
        </>
    )
}
