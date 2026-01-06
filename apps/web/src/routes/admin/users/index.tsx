import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Search01Icon,
    Cancel01Icon,
    ViewIcon,
} from '@hugeicons/core-free-icons'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { Input } from '@/components/ui/input'
import { PageHeader } from '@/components/shell'
import { ColumnDef } from '@tanstack/react-table'
import { useAdminUsers } from '@/hooks'
import type { AdminUser } from '@/lib/connect'
import { formatDate } from '@/lib/utils'
import { Link } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/users/')({
    component: AdminUsersPage,
})

const columns: ColumnDef<AdminUser>[] = [
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
        accessorKey: 'role',
        header: 'Role',
        cell: ({ row }) => (
            <Badge variant={row.original.role === 'admin' ? 'default' : 'outline'} className="text-xs">
                {row.original.role}
            </Badge>
        ),
    },
    {
        accessorKey: 'isActive',
        header: 'Status',
        cell: ({ row }) => {
            const isSuspended = row.original.isSuspended
            const isFlagged = row.original.isFlagged

            if (isSuspended) {
                return <Badge variant="destructive" className="text-xs">Suspended</Badge>
            }
            if (isFlagged) {
                return <Badge variant="warning" className="text-xs">Flagged</Badge>
            }
            return <Badge variant="success" className="text-xs">Active</Badge>
        },
    },
    {
        accessorKey: 'createdAt',
        header: 'Joined',
        cell: ({ row }) => {
            const ts = row.original.createdAt
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
                    <Link to="/admin/users/$userId" params={{ userId: row.original.id }}>
                        <HugeiconsIcon icon={ViewIcon} size={14} />
                    </Link>
                </Button>
            </div>
        ),
    },
]

type QuickFilter = 'all' | 'active' | 'flagged' | 'suspended'

function AdminUsersPage() {
    const [pagination, setPagination] = useState({
        pageIndex: 0,
        pageSize: 20,
    })
    const [quickFilter, setQuickFilter] = useState<QuickFilter>('all')
    const [searchQuery, setSearchQuery] = useState('')

    const { data, isLoading, isError, error } = useAdminUsers({
        pageSize: pagination.pageSize,
        offset: pagination.pageIndex * pagination.pageSize,
    })

    // Filter users based on quick filter and search
    const filteredUsers = data?.users?.filter((user) => {
        // Quick filter
        if (quickFilter === 'active' && (user.isSuspended || user.isFlagged)) return false
        if (quickFilter === 'flagged' && !user.isFlagged) return false
        if (quickFilter === 'suspended' && !user.isSuspended) return false

        // Search filter
        if (searchQuery) {
            const query = searchQuery.toLowerCase()
            return (
                user.email.toLowerCase().includes(query) ||
                user.name.toLowerCase().includes(query)
            )
        }
        return true
    }) ?? []

    const totalCount = data?.totalCount || 0
    const pageCount = Math.ceil(totalCount / pagination.pageSize)

    return (
        <>
            <PageHeader
                title="User Management"
                description="View and manage user accounts"
            />

            {/* Filters */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
                {/* Quick Filters */}
                <div className="flex items-center gap-2">
                    {(['all', 'active', 'flagged', 'suspended'] as const).map((filter) => {
                        const isActive = quickFilter === filter
                        return (
                            <button
                                key={filter}
                                onClick={() => setQuickFilter(filter)}
                                className={`
                                    h-8 px-4 rounded-md text-xs font-medium border transition-all duration-200
                                    ${isActive
                                        ? 'border-solid border-border bg-secondary text-foreground shadow-sm'
                                        : 'border-dashed border-border text-muted-foreground hover:text-foreground hover:bg-secondary/50 bg-transparent'
                                    }
                                `}
                            >
                                {filter.charAt(0).toUpperCase() + filter.slice(1)}
                            </button>
                        )
                    })}
                </div>

                {/* Search */}
                <div className="relative max-w-xs">
                    <HugeiconsIcon
                        icon={Search01Icon}
                        size={14}
                        className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                    />
                    <Input
                        placeholder="Search by email or name..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="pl-9 h-8 text-xs"
                    />
                    {searchQuery && (
                        <button
                            onClick={() => setSearchQuery('')}
                            className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                        >
                            <HugeiconsIcon icon={Cancel01Icon} size={12} />
                        </button>
                    )}
                </div>
            </div>

            {/* Users Table */}
            <div className="rounded-xl border border-border/40 bg-card shadow-sm overflow-hidden">
                {isError ? (
                    <div className="p-8 text-center text-sm text-destructive">
                        Failed to load users. {error?.message}
                    </div>
                ) : (
                    <DataTable
                        columns={columns}
                        data={filteredUsers}
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
                {totalCount} total users
            </div>
        </>
    )
}
