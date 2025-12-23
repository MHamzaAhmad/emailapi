import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Add01Icon,
    ArrowRight01Icon,
    RefreshIcon,
    Delete01Icon,
} from '@hugeicons/core-free-icons'
import { Shell, PageHeader } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
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
import type { DomainStatus, Domain } from '@/types'

export const Route = createFileRoute('/domains/')(
    {
        component: DomainsPage,
    }
)

function DomainsPage() {
    return (
        <AuthGuard>
            <DomainsContent />
        </AuthGuard>
    )
}

function getStatusBadge(status: DomainStatus) {
    switch (status) {
        case 'ready':
            return <Badge variant="success">ready</Badge>
        case 'verifying':
            return <Badge variant="secondary">verifying</Badge>
        case 'pending':
            return <Badge variant="warning">pending</Badge>
        case 'failed':
            return <Badge variant="destructive">failed</Badge>
        case 'degraded':
            return <Badge variant="warning">degraded</Badge>
        default:
            return <Badge variant="outline">{status}</Badge>
    }
}

function DomainsContent() {
    const [isAddOpen, setIsAddOpen] = useState(false)
    const [newDomain, setNewDomain] = useState('')

    const { data: domains, isLoading } = useDomains()
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

    return (
        <Shell>
            <PageHeader
                title="Domains"
                description="Manage your sending domains"
                actions={
                    <Dialog open={isAddOpen} onOpenChange={setIsAddOpen}>
                        <DialogTrigger asChild>
                            <Button size="sm">
                                <HugeiconsIcon icon={Add01Icon} size={12} strokeWidth={1.5} />
                                Add domain
                            </Button>
                        </DialogTrigger>
                        <DialogContent>
                            <DialogHeader>
                                <DialogTitle>Add Domain</DialogTitle>
                                <DialogDescription>
                                    Enter your root domain without subdomains.
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
                                    />
                                </div>
                            </div>
                            <DialogFooter>
                                <Button variant="outline" onClick={() => setIsAddOpen(false)}>
                                    Cancel
                                </Button>
                                <Button
                                    onClick={handleAdd}
                                    disabled={!newDomain.trim() || addMutation.isPending}
                                >
                                    {addMutation.isPending ? 'Adding...' : 'Add'}
                                </Button>
                            </DialogFooter>
                        </DialogContent>
                    </Dialog>
                }
            />

            <Card>
                <CardContent className="p-0">
                    {isLoading ? (
                        <div className="p-4 text-center text-xs text-muted-foreground">
                            Loading...
                        </div>
                    ) : !domains?.length ? (
                        <div className="p-4 text-center text-xs text-muted-foreground">
                            No domains yet. Add one to get started.
                        </div>
                    ) : (
                        <div className="divide-y divide-border">
                            {domains.map((domain: Domain) => (
                                <div key={domain.id} className="flex items-center justify-between px-3 py-2.5">
                                    <div className="flex items-center gap-3">
                                        <div>
                                            <div className="text-xs font-medium">{domain.domain}</div>
                                            <div className="text-2xs text-muted-foreground">
                                                {domain.summary?.message || `Region: ${domain.region}`}
                                            </div>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        {getStatusBadge(domain.status)}
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            onClick={() => verifyMutation.mutate(domain.id)}
                                            disabled={verifyMutation.isPending}
                                            title="Verify"
                                        >
                                            <HugeiconsIcon
                                                icon={RefreshIcon}
                                                size={14}
                                                strokeWidth={1.5}
                                                className={verifyMutation.isPending ? 'animate-spin' : ''}
                                            />
                                        </Button>
                                        <Link
                                            to="/domains/$domainId"
                                            params={{ domainId: domain.id }}
                                        >
                                            <Button variant="ghost" size="icon" title="View DNS">
                                                <HugeiconsIcon icon={ArrowRight01Icon} size={14} strokeWidth={1.5} />
                                            </Button>
                                        </Link>
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            onClick={() => handleDelete(domain.id)}
                                            disabled={deleteMutation.isPending}
                                            title="Delete"
                                            className="text-destructive hover:text-destructive"
                                        >
                                            <HugeiconsIcon icon={Delete01Icon} size={14} strokeWidth={1.5} />
                                        </Button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </CardContent>
            </Card>
        </Shell>
    )
}
