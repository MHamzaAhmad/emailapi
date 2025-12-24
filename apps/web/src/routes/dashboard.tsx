import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    FilterHorizontalIcon,
    Cancel01Icon,
} from '@hugeicons/core-free-icons'
import { DateRange } from "react-day-picker"
import { DatePickerWithRange } from "@/components/ui/date-range-picker"
import { Shell } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { Label } from '@/components/ui/label'
import {
    Popover,
    PopoverContent,
    PopoverTrigger,
} from '@/components/ui/popover'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { ColumnDef } from "@tanstack/react-table"
import { useActivityLogs } from '@/hooks'
import { ActivityLog } from '@/types'
import { formatDate } from '@/lib/utils'
import { Timestamp } from '@bufbuild/protobuf/wkt'

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
        cell: ({ row }) => {
            const ts = row.getValue("timestamp") as Timestamp | undefined
            if (!ts) return <span className="text-xs text-muted-foreground whitespace-nowrap">-</span>
            const date = new Date(Number(ts.seconds) * 1000 + Math.round(ts.nanos / 1e6))
            return <span className="text-xs text-muted-foreground whitespace-nowrap">{formatDate(date)}</span>
        },
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

// Filter options
const entityTypes = [
    { value: 'all', label: 'All Types' },
    { value: 'email', label: 'Email' },
    { value: 'domain', label: 'Domain' },
    { value: 'api_key', label: 'API Key' },
    { value: 'webhook', label: 'Webhook' },
]

const actions = [
    { value: 'all', label: 'All Actions' },
    { value: 'sent', label: 'Sent' },
    { value: 'delivered', label: 'Delivered' },
    { value: 'bounced', label: 'Bounced' },
    { value: 'complaint', label: 'Complaint' },
    { value: 'created', label: 'Created' },
    { value: 'updated', label: 'Updated' },
    { value: 'deleted', label: 'Deleted' },
    { value: 'verified', label: 'Verified' },
]

interface AdvancedFilters {
    entityType: string
    action: string
    dateRange: DateRange | undefined
}

function DashboardContent() {
    const [pagination, setPagination] = useState({
        pageIndex: 0,
        pageSize: 10,
    })
    const [quickFilter, setQuickFilter] = useState<'all' | 'emails' | 'bounced' | 'complaints'>('all')
    const [advancedFilters, setAdvancedFilters] = useState<AdvancedFilters>({
        entityType: 'all',
        action: 'all',
        dateRange: undefined,
    })
    const [isFiltersOpen, setIsFiltersOpen] = useState(false)

    // Check if any advanced filters are active
    const hasAdvancedFilters = (advancedFilters.entityType !== 'all' && advancedFilters.entityType !== '') ||
        (advancedFilters.action !== 'all' && advancedFilters.action !== '') ||
        advancedFilters.dateRange?.from !== undefined

    const activeFilterCount = [
        advancedFilters.entityType !== 'all' && advancedFilters.entityType !== '' ? true : false,
        advancedFilters.action !== 'all' && advancedFilters.action !== '' ? true : false,
        advancedFilters.dateRange?.from !== undefined
    ].filter(Boolean).length

    // Build query params
    const queryParams: any = {
        pageSize: pagination.pageSize,
        offset: pagination.pageIndex * pagination.pageSize,
    }

    // Advanced filters take precedence over quick filters
    if (hasAdvancedFilters) {
        if (advancedFilters.entityType && advancedFilters.entityType !== 'all') queryParams.entityType = advancedFilters.entityType
        if (advancedFilters.action && advancedFilters.action !== 'all') queryParams.action = advancedFilters.action
        if (advancedFilters.dateRange?.from) {
            queryParams.startTime = BigInt(advancedFilters.dateRange.from.getTime())
        }
        if (advancedFilters.dateRange?.to) {
            // Set to end of day
            const endDate = new Date(advancedFilters.dateRange.to)
            endDate.setHours(23, 59, 59, 999)
            queryParams.endTime = BigInt(endDate.getTime())
        }
    } else {
        // Quick filters
        if (quickFilter === 'emails') {
            queryParams.entityType = 'email'
        } else if (quickFilter === 'bounced') {
            queryParams.entityType = 'email'
            queryParams.action = 'bounced'
        } else if (quickFilter === 'complaints') {
            queryParams.entityType = 'email'
            queryParams.action = 'complaint'
        }
    }

    const { data, isLoading, isError, error } = useActivityLogs(queryParams)

    const totalCount = data?.totalCount || 0
    const pageCount = Math.ceil(totalCount / pagination.pageSize)
    const showLoading = isLoading && !data

    const handleApplyFilters = () => {
        setPagination(p => ({ ...p, pageIndex: 0 }))
        setIsFiltersOpen(false)
    }

    const handleClearFilters = () => {
        setAdvancedFilters({ entityType: 'all', action: 'all', dateRange: undefined })
        setQuickFilter('all')
        setPagination(p => ({ ...p, pageIndex: 0 }))
    }

    return (
        <Shell>
            {/* Header / Filters Section */}
            <div className="flex flex-col gap-6 mb-8 mt-2">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    {/* Quick Filters */}
                    <div className="flex items-center gap-3">
                        {[
                            { id: 'all', label: 'All' },
                            { id: 'emails', label: 'Emails' },
                            { id: 'bounced', label: 'Bounced' },
                            { id: 'complaints', label: 'Complaints' }
                        ].map((item) => {
                            const isActive = quickFilter === item.id && !hasAdvancedFilters
                            return (
                                <button
                                    key={item.id}
                                    onClick={() => {
                                        setQuickFilter(item.id as any)
                                        setAdvancedFilters({ entityType: 'all', action: 'all', dateRange: undefined })
                                        setPagination(p => ({ ...p, pageIndex: 0 }))
                                    }}
                                    className={`
                                        h-8 px-4 rounded-md text-xs font-medium border transition-all duration-200 relative select-none
                                        ${isActive
                                            ? 'border-solid border-border bg-secondary text-foreground shadow-sm'
                                            : 'border-dashed border-border text-muted-foreground hover:border-border hover:bg-secondary/50 hover:text-foreground bg-transparent'
                                        }
                                    `}
                                >
                                    {item.label}
                                </button>
                            )
                        })}
                    </div>

                    {/* Right Side Actions */}
                    <div className="flex items-center gap-2">
                        {hasAdvancedFilters && (
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={handleClearFilters}
                                className="h-8 gap-1.5 text-xs text-muted-foreground hover:text-foreground"
                            >
                                <HugeiconsIcon icon={Cancel01Icon} size={12} />
                                Clear filters
                            </Button>
                        )}
                        <Popover open={isFiltersOpen} onOpenChange={setIsFiltersOpen}>
                            <PopoverTrigger asChild>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    className={`h-8 gap-2 text-xs font-medium shadow-sm ${hasAdvancedFilters ? 'border-primary bg-primary/5 text-primary' : 'bg-background hover:bg-muted/50 border-dashed'}`}
                                >
                                    <HugeiconsIcon icon={FilterHorizontalIcon} size={14} />
                                    <span>More Filters</span>
                                    {activeFilterCount > 0 && (
                                        <span className="ml-1 flex h-4 w-4 items-center justify-center rounded-full bg-primary text-[10px] text-primary-foreground font-semibold">
                                            {activeFilterCount}
                                        </span>
                                    )}
                                </Button>
                            </PopoverTrigger>
                            <PopoverContent className="w-[340px] p-0" align="end">
                                <div className="p-4 space-y-4">
                                    <div className="grid grid-cols-2 gap-4">
                                        {/* Entity Type */}
                                        <div className="space-y-1.5">
                                            <Label className="text-xs font-medium text-muted-foreground">Entity Type</Label>
                                            <Select
                                                value={advancedFilters.entityType}
                                                onValueChange={(value) => setAdvancedFilters(f => ({ ...f, entityType: value }))}
                                            >
                                                <SelectTrigger variant="dashed" size="lg" className="w-full text-xs shadow-sm">
                                                    <SelectValue placeholder="All Types" />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    {entityTypes.map((type) => (
                                                        <SelectItem key={type.value} value={type.value} className="text-xs">
                                                            {type.label}
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                        </div>

                                        {/* Action */}
                                        <div className="space-y-1.5">
                                            <Label className="text-xs font-medium text-muted-foreground">Action</Label>
                                            <Select
                                                value={advancedFilters.action}
                                                onValueChange={(value) => setAdvancedFilters(f => ({ ...f, action: value }))}
                                            >
                                                <SelectTrigger variant="dashed" size="lg" className="w-full text-xs shadow-sm">
                                                    <SelectValue placeholder="All Actions" />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    {actions.map((action) => (
                                                        <SelectItem key={action.value} value={action.value} className="text-xs">
                                                            {action.label}
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                        </div>
                                    </div>

                                    {/* Date Range */}
                                    <div className="space-y-1.5">
                                        <Label className="text-xs font-medium text-muted-foreground">Date Range</Label>
                                        <DatePickerWithRange
                                            date={advancedFilters.dateRange}
                                            setDate={(date) => setAdvancedFilters(f => ({ ...f, dateRange: date }))}
                                        />
                                    </div>
                                </div>
                                <div className="p-3 bg-muted/30 border-t border-border flex items-center justify-between">
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={handleClearFilters}
                                        className="h-7 text-xs hover:bg-transparent hover:text-foreground text-muted-foreground px-2"
                                    >
                                        Reset
                                    </Button>
                                    <Button
                                        size="sm"
                                        onClick={handleApplyFilters}
                                        className="h-7 text-xs px-4"
                                    >
                                        Apply Filters
                                    </Button>
                                </div>
                            </PopoverContent>
                        </Popover>
                    </div>
                </div>
            </div>

            {/* Content Table */}
            <div className="space-y-4">
                <div className="rounded-xl border border-border/40 bg-card shadow-sm overflow-hidden">
                    {isError ? (
                        <div className="p-8 text-center text-sm text-destructive">
                            Failed to load activity logs. {error?.message}
                        </div>
                    ) : (
                        <DataTable
                            columns={columns}
                            data={data?.logs || []}
                            isLoading={showLoading}
                            manualPagination={true}
                            pageCount={pageCount}
                            pageIndex={pagination.pageIndex}
                            pageSize={pagination.pageSize}
                            onPaginationChange={setPagination}
                        />
                    )}
                </div>
            </div>

            <div className="mt-4 text-xs text-muted-foreground/60 text-right">
                Displaying most recent events from real-time stream
            </div>
        </Shell>
    )
}
