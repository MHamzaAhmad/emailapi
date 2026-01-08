import { createFileRoute, Link } from '@tanstack/react-router'
import { useState, useMemo } from 'react'
import { useUser } from '@clerk/clerk-react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Add01Icon,
} from '@hugeicons/core-free-icons'
import { ColumnDef } from '@tanstack/react-table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { DataTable } from '@/components/ui/data-table'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'
import { useDomains, useAddDomain, useDeleteDomain, useVerifyDomain } from '@/hooks'
import { DomainStatus } from '@/generated/v1/domain_pb'
import type { Domain } from '@/generated/v1/domain_pb'

export const Route = createFileRoute('/_authed/domains/')({
    component: DomainsContent,
})

function getStatusBadge(status: DomainStatus) {
    const className = "text-[10px] px-2 py-0.5 uppercase tracking-wider font-medium"
    switch (status) {
        case DomainStatus.READY:
            return <Badge variant="success" className={className}>Ready</Badge>
        case DomainStatus.VERIFYING:
            return <Badge variant="secondary" className={className}>Verifying</Badge>
        case DomainStatus.PENDING:
            return <Badge variant="warning" className={className}>Pending</Badge>
        case DomainStatus.FAILED:
            return <Badge variant="destructive" className={className}>Failed</Badge>
        case DomainStatus.DEGRADED:
            return <Badge variant="warning" className={className}>Degraded</Badge>
        default:
            return <Badge variant="outline" className={className}>Unknown</Badge>
    }
}

function DomainsContent() {
    const { user } = useUser()
    const [isAddOpen, setIsAddOpen] = useState(false)
    const [newDomain, setNewDomain] = useState('')

    const [pagination, setPagination] = useState({
        pageIndex: 0,
        pageSize: 10,
    })

    const { data: domainsResponse, isLoading } = useDomains(pagination.pageIndex + 1, pagination.pageSize)
    const domains = domainsResponse?.data || []
    const total = domainsResponse?.total || 0
    const pageCount = Math.ceil(total / pagination.pageSize)

    const addMutation = useAddDomain()
    const deleteMutation = useDeleteDomain()
    const verifyMutation = useVerifyDomain()

    const handleAdd = async () => {
        if (!newDomain.trim()) return

        try {
            await addMutation.mutateAsync({ domain: newDomain.trim() })
            setNewDomain('')
            setIsAddOpen(false)
        } catch (err) {
            console.error('Failed to add domain:', err)
        }
    }

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure you want to delete this domain?')) return
        await deleteMutation.mutateAsync(id)
    }

    const columns = useMemo<ColumnDef<Domain>[]>(
        () => [
            {
                accessorKey: 'domain',
                header: 'Domain',
                cell: ({ row }) => (
                    <div>
                        <div className="font-medium text-xs text-foreground">{row.original.domain}</div>
                        {row.original.summary?.message && (
                            <div className="text-[10px] text-muted-foreground line-clamp-1 mt-0.5">
                                {row.original.summary.message}
                            </div>
                        )}
                    </div>
                ),
            },
            {
                accessorKey: 'region',
                header: 'Region',
                cell: ({ row }) => (
                    <div className="text-xs text-muted-foreground">{row.original.region}</div>
                ),
            },
            {
                accessorKey: 'status',
                header: 'Status',
                cell: ({ row }) => getStatusBadge(row.original.status),
            },
            {
                id: 'actions',
                cell: ({ row }) => (
                    <div className="flex items-center justify-end gap-2">
                        <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 text-xs font-medium"
                            onClick={() => verifyMutation.mutate(row.original.id)}
                            disabled={verifyMutation.isPending}
                        >
                            {verifyMutation.isPending ? 'Verifying...' : 'Verify'}
                        </Button>
                        <Link
                            to="/domains/$domainId"
                            params={{ domainId: row.original.id }}
                        >
                            <Button variant="ghost" size="sm" className="h-7 text-xs font-medium">
                                View DNS
                            </Button>
                        </Link>
                        <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 text-xs font-medium text-destructive hover:text-destructive hover:bg-destructive/10"
                            onClick={() => handleDelete(row.original.id)}
                            disabled={deleteMutation.isPending}
                        >
                            Delete
                        </Button>
                    </div>
                ),
            },
        ],
        [verifyMutation, deleteMutation]
    )

    const userEmail = user?.primaryEmailAddress?.emailAddress || 'your-email@example.com'

    return (
        <>
            {/* Header Actions */}
            {/* Header Actions */}
            <div className="flex flex-col gap-6 mb-8 mt-2">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <div className="text-xs text-muted-foreground">
                        You can use the free domain <code className="bg-muted/50 px-1.5 py-0.5 rounded font-mono text-[10px] mx-1 text-foreground">yourname@sandbox.simpleemailapi.dev</code> to test, but you can only send to your own email: <span className="font-medium text-foreground">{userEmail}</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Dialog open={isAddOpen} onOpenChange={setIsAddOpen}>
                            <DialogTrigger asChild>
                                <Button size="lg" className="h-8 shadow-sm">
                                    <HugeiconsIcon icon={Add01Icon} size={14} strokeWidth={2} />
                                    Add domain
                                </Button>
                            </DialogTrigger>
                            <DialogContent>
                                <DialogHeader>
                                    <DialogTitle>Add Domain</DialogTitle>
                                    <DialogDescription>
                                        Enter your root domain or a subdomain.
                                    </DialogDescription>
                                </DialogHeader>
                                <div className="space-y-3">
                                    <div className="space-y-1.5">
                                        <Label htmlFor="domain">Domain</Label>
                                        <Input
                                            id="domain"
                                            placeholder="example.com"
                                            value={newDomain}
                                            onChange={(e) => setNewDomain(e.target.value)}
                                            className="h-9"
                                        />
                                    </div>
                                </div>
                                <DialogFooter>
                                    <Button variant="outline" onClick={() => setIsAddOpen(false)} className="h-8">
                                        Cancel
                                    </Button>
                                    <Button
                                        onClick={handleAdd}
                                        disabled={!newDomain.trim() || addMutation.isPending}
                                        className="h-8"
                                    >
                                        {addMutation.isPending ? 'Adding...' : 'Add'}
                                    </Button>
                                </DialogFooter>
                            </DialogContent>
                        </Dialog>
                    </div>
                </div>
            </div>

            <div className="rounded-xl border border-border/40 bg-card shadow-sm overflow-hidden">
                <DataTable
                    columns={columns}
                    data={domains}
                    isLoading={isLoading}
                    manualPagination={true}
                    pageCount={pageCount}
                    pageIndex={pagination.pageIndex}
                    pageSize={pagination.pageSize}
                    onPaginationChange={setPagination}
                />
            </div>
        </>
    )
}
