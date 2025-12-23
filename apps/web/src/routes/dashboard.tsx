import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    PlusSignIcon,
    FilterHorizontalIcon,
} from '@hugeicons/core-free-icons'
import { Shell } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { ColumnDef } from "@tanstack/react-table"
import { useActivityLogs } from '@/hooks'
import { ActivityLog } from '@/types'
import { formatDate } from '@/lib/utils'

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

const columns: ColumnDef<ActivityLog>[] = [
    {
        accessorKey: "timestamp",
        header: "Time",
        cell: ({ row }) => <span className="text-xs text-muted-foreground whitespace-nowrap">{formatDate(row.getValue("timestamp"))}</span>,
    },
    {
        accessorKey: "entityType",
        header: "Type",
        cell: ({ row }) => <span className="capitalize font-medium text-xs">{(row.getValue("entityType") as string).replace('_', ' ')}</span>,
    },
    {
        accessorKey: "entityId",
        header: "Entity ID",
        cell: ({ row }) => <span className="font-mono text-xs text-muted-foreground">{row.getValue("entityId")}</span>,
    },
    {
        accessorKey: "action",
        header: "Action",
        cell: ({ row }) => <span className="capitalize text-xs font-medium">{row.getValue("action")}</span>,
    },
    {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => {
            const status = row.getValue("status") as string
            const getVariant = (s: string) => {
                switch (s) {
                    case 'success':
                    case 'delivered':
                        return 'success'
                    case 'failed':
                    case 'bounced':
                        return 'destructive'
                    case 'pending':
                        return 'warning'
                    default:
                        return 'outline'
                }
            }
            return (
                <Badge variant={getVariant(status)} className="text-[10px] px-2 py-0 uppercase tracking-wider">
                    {status}
                </Badge>
            )
        },
    },
    {
        accessorKey: "details",
        header: "Details",
        cell: ({ row }) => <span className="text-xs text-muted-foreground truncate max-w-[200px] block" title={row.getValue("details")}>{row.getValue("details")}</span>,
    },
]

function DashboardContent() {
    const [pagination, setPagination] = useState({
        pageIndex: 0,
        pageSize: 10,
    })

    const { data, isLoading } = useActivityLogs({
        pageSize: pagination.pageSize,
        offset: pagination.pageIndex * pagination.pageSize,
    })

    const totalCount = data?.totalCount || 0
    const pageCount = Math.ceil(totalCount / pagination.pageSize)

    return (
        <Shell>
            {/* Header Area */}
            <div className="flex flex-col gap-6 mb-8">
                <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                    <div className="flex items-center gap-2 text-2xl font-bold tracking-tight">
                        <h1>Dashboard</h1>
                        <span className="text-muted-foreground font-light">/</span>
                        <h1 className="text-foreground">Activity</h1>
                    </div>
                    <div className="flex items-center gap-2">
                        <Button variant="outline" size="sm" className="h-9 gap-2 bg-background hover:bg-muted/50 border-input/60 shadow-sm">
                            <HugeiconsIcon icon={FilterHorizontalIcon} size={14} />
                            <span>Filters</span>
                        </Button>
                        <Button size="sm" className="h-9 gap-2 shadow-sm font-medium">
                            <HugeiconsIcon icon={PlusSignIcon} size={14} />
                            <span>Dispatch Email</span>
                        </Button>
                    </div>
                </div>
            </div>

            {/* Main Content - Data Table */}
            <div className="space-y-4">
                <DataTable
                    columns={columns}
                    data={data?.logs || []}
                    isLoading={isLoading}
                    manualPagination={true}
                    pageCount={pageCount}
                    pageIndex={pagination.pageIndex}
                    pageSize={pagination.pageSize}
                    onPaginationChange={setPagination}
                    searchKey="entityId"
                />
            </div>

            <div className="mt-4 text-xs text-muted-foreground">
                Showing {data?.logs.length || 0} of {totalCount} events
            </div>
        </Shell>
    )
}
