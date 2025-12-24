import { createFileRoute } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Add01Icon,
    Copy01Icon,
    AlertCircleIcon,
    Tick02Icon,
} from '@hugeicons/core-free-icons'
import { Shell } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { useState, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { DataTable } from '@/components/ui/data-table'
import { ColumnDef } from '@tanstack/react-table'
import { useAPIKeys, useDeleteAPIKey, useRevokeAPIKey } from '@/hooks'
import { ApiKey, Scope } from '@/generated/v1/apikey_pb'
import { formatDate } from '@/lib/utils'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useCreateAPIKey } from '@/hooks'

export const Route = createFileRoute('/api-keys')({
    component: APIKeysPage,
})

function APIKeysPage() {
    return (
        <AuthGuard>
            <APIKeysContent />
        </AuthGuard>
    )
}

function CopyButton({ text }: { text: string }) {
    const [copied, setCopied] = useState(false)

    const handleCopy = async () => {
        await navigator.clipboard.writeText(text)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
    }

    return (
        <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6 ml-1 flex-shrink-0 text-muted-foreground hover:text-foreground"
            onClick={handleCopy}
            title="Copy value"
        >
            <HugeiconsIcon
                icon={copied ? Tick02Icon : Copy01Icon}
                size={12}
                strokeWidth={1.5}
                className={copied ? "text-emerald-500" : ""}
            />
        </Button>
    )
}

// Available scopes for API keys
const AVAILABLE_SCOPES = [
    { value: Scope.EMAIL_SEND, label: 'Email Send', description: 'Send emails' },
    { value: Scope.EMAIL_READ, label: 'Email Read', description: 'Read email data' },
    { value: Scope.DOMAIN_READ, label: 'Domain Read', description: 'View domains' },
    { value: Scope.DOMAIN_WRITE, label: 'Domain Write', description: 'Manage domains' },
] as const;

// Helper to get scope label
const getScopeLabel = (scope: Scope): string => {
    switch (scope) {
        case Scope.EMAIL_SEND: return 'email:send';
        case Scope.EMAIL_READ: return 'email:read';
        case Scope.DOMAIN_READ: return 'domain:read';
        case Scope.DOMAIN_WRITE: return 'domain:write';
        case Scope.APIKEY_READ: return 'apikey:read';
        case Scope.APIKEY_WRITE: return 'apikey:write';
        case Scope.USER_READ: return 'user:read';
        case Scope.USER_WRITE: return 'user:write';
        default: return 'unknown';
    }
};

function APIKeysContent() {
    const [pagination, setPagination] = useState({
        pageIndex: 0,
        pageSize: 10,
    })

    // Create Key Dialog State
    const [isCreateOpen, setIsCreateOpen] = useState(false)
    const [newKeyName, setNewKeyName] = useState('')
    const [selectedScopes, setSelectedScopes] = useState<Scope[]>([Scope.EMAIL_SEND, Scope.DOMAIN_READ])
    const [createdKey, setCreatedKey] = useState<{ key: ApiKey, raw: string } | null>(null)

    const { data: apiKeysResponse, isLoading } = useAPIKeys(pagination.pageIndex + 1, pagination.pageSize)
    const apiKeys = apiKeysResponse?.data || []
    const total = apiKeysResponse?.total || 0
    const pageCount = Math.ceil(total / pagination.pageSize)

    const createMutation = useCreateAPIKey()
    const deleteMutation = useDeleteAPIKey()
    const revokeMutation = useRevokeAPIKey()

    const toggleScope = (scope: Scope) => {
        setSelectedScopes(prev =>
            prev.includes(scope)
                ? prev.filter(s => s !== scope)
                : [...prev, scope]
        )
    }

    const handleCreate = async () => {
        if (!newKeyName.trim()) return
        try {
            const response = await createMutation.mutateAsync({
                name: newKeyName,
                scopes: selectedScopes,
            })
            setCreatedKey({ key: response.apiKey!, raw: response.rawKey })
            setNewKeyName('')
            setSelectedScopes([Scope.EMAIL_SEND, Scope.DOMAIN_READ])
            // Don't close dialog yet, show the raw key
        } catch (err) {
            console.error('Failed to create API key:', err)
        }
    }

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure you want to delete this API key? This action cannot be undone.')) return
        await deleteMutation.mutateAsync(id)
    }

    const handleRevoke = async (id: string) => {
        if (!confirm('Are you sure you want to revoke this API key? It will stop working immediately.')) return
        await revokeMutation.mutateAsync(id)
    }

    const columns = useMemo<ColumnDef<ApiKey>[]>(
        () => [
            {
                accessorKey: 'name',
                header: 'Name',
                cell: ({ row }) => (
                    <div className="font-medium text-xs text-foreground">{row.original.name}</div>
                ),
            },
            {
                accessorKey: 'keyPrefix',
                header: 'Key Prefix',
                cell: ({ row }) => (
                    <div className="flex items-center gap-2 font-mono text-xs text-muted-foreground group">
                        <span className="bg-muted px-1.5 py-0.5 rounded text-foreground/80">
                            {row.original.keyPrefix}
                        </span>
                    </div>
                ),
            },
            {
                accessorKey: 'scopes',
                header: 'Scopes',
                cell: ({ row }) => (
                    <div className="flex flex-wrap gap-1">
                        {row.original.scopes.slice(0, 2).map((scope) => (
                            <Badge key={scope} variant="outline" className="text-[10px] px-1.5 py-0">
                                {getScopeLabel(scope)}
                            </Badge>
                        ))}
                        {row.original.scopes.length > 2 && (
                            <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                                +{row.original.scopes.length - 2}
                            </Badge>
                        )}
                    </div>
                ),
            },
            {
                accessorKey: 'createdAt',
                header: 'Created',
                cell: ({ row }) => (
                    <div className="text-xs text-muted-foreground">
                        {row.original.createdAt ? formatDate(new Date(Number(row.original.createdAt.seconds) * 1000)) : '-'}
                    </div>
                ),
            },
            {
                accessorKey: 'lastUsedAt',
                header: 'Last Used',
                cell: ({ row }) => (
                    <div className="text-xs text-muted-foreground">
                        {row.original.lastUsedAt ? formatDate(new Date(Number(row.original.lastUsedAt.seconds) * 1000)) : 'Never'}
                    </div>
                ),
            },
            {
                id: 'actions',
                cell: ({ row }) => (
                    <div className="flex items-center justify-end gap-2">
                        {row.original.isActive ? (
                            <Button
                                variant="ghost"
                                size="sm"
                                className="h-7 text-xs font-medium text-amber-600 hover:text-amber-700 hover:bg-amber-50"
                                onClick={() => handleRevoke(row.original.id)}
                                disabled={revokeMutation.isPending}
                            >
                                Revoke
                            </Button>
                        ) : (
                            <Badge variant="outline" className="text-[10px] text-muted-foreground">Revoked</Badge>
                        )}
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
        [revokeMutation, deleteMutation]
    )

    return (
        <Shell>
            <div className="flex flex-col gap-6 mb-8 mt-2">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <div /> {/* Spacer for alignment */}
                    <div className="flex items-center gap-2">
                        <Dialog open={isCreateOpen} onOpenChange={(open) => {
                            if (!open) setCreatedKey(null)
                            setIsCreateOpen(open)
                        }}>
                            <DialogTrigger asChild>
                                <Button size="lg" className="h-8 shadow-sm">
                                    <HugeiconsIcon icon={Add01Icon} size={14} strokeWidth={2} className="mr-1.5" />
                                    Create Key
                                </Button>
                            </DialogTrigger>
                            <DialogContent>
                                <DialogHeader>
                                    <DialogTitle>Create API Key</DialogTitle>
                                    <DialogDescription>
                                        Create a new API key for your applications.
                                    </DialogDescription>
                                </DialogHeader>

                                {createdKey ? (
                                    <div className="space-y-4">
                                        <div className="bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-900 rounded-md p-3 flex gap-3 text-sm text-amber-900 dark:text-amber-200">
                                            <HugeiconsIcon icon={AlertCircleIcon} size={16} className="shrink-0 mt-0.5" />
                                            <div>
                                                <p className="font-medium">Save this key now</p>
                                                <p className="mt-1 opacity-90">This is the only time the full API key will be shown. You won't be able to see it again.</p>
                                            </div>
                                        </div>
                                        <div className="space-y-1.5">
                                            <Label>API Key</Label>
                                            <div className="flex items-center gap-2">
                                                <div className="flex-1 bg-muted p-2 rounded-md font-mono text-sm break-all">
                                                    {createdKey.raw}
                                                </div>
                                                <CopyButton text={createdKey.raw} />
                                            </div>
                                        </div>
                                        <DialogFooter>
                                            <Button onClick={() => setIsCreateOpen(false)}>Done</Button>
                                        </DialogFooter>
                                    </div>
                                ) : (
                                    <>
                                        <div className="space-y-4 py-2">
                                            <div className="space-y-1.5">
                                                <Label htmlFor="name">Name</Label>
                                                <Input
                                                    id="name"
                                                    placeholder="e.g. Production Server"
                                                    value={newKeyName}
                                                    onChange={(e) => setNewKeyName(e.target.value)}
                                                />
                                            </div>
                                            <div className="space-y-2">
                                                <Label>Scopes</Label>
                                                <div className="flex flex-wrap gap-2">
                                                    {AVAILABLE_SCOPES.map((scope) => {
                                                        const isActive = selectedScopes.includes(scope.value)
                                                        return (
                                                            <button
                                                                key={scope.value}
                                                                type="button"
                                                                onClick={() => toggleScope(scope.value)}
                                                                className={`
                                                                    h-8 px-3 rounded-md text-xs font-medium border transition-all duration-200 select-none
                                                                    ${isActive
                                                                        ? 'border-solid border-primary bg-primary/10 text-primary shadow-sm'
                                                                        : 'border-dashed border-border text-muted-foreground hover:border-border hover:bg-secondary/50 hover:text-foreground bg-transparent'
                                                                    }
                                                                `}
                                                            >
                                                                {scope.label}
                                                            </button>
                                                        )
                                                    })}
                                                </div>
                                                <p className="text-[10px] text-muted-foreground">Select the permissions for this API key</p>
                                            </div>
                                        </div>
                                        <DialogFooter>
                                            <Button variant="outline" onClick={() => setIsCreateOpen(false)}>Cancel</Button>
                                            <Button onClick={handleCreate} disabled={!newKeyName.trim() || createMutation.isPending}>
                                                {createMutation.isPending ? 'Creating...' : 'Create'}
                                            </Button>
                                        </DialogFooter>
                                    </>
                                )}
                            </DialogContent>
                        </Dialog>
                    </div>
                </div>
            </div>

            <div className="rounded-xl border border-border/40 bg-card shadow-sm overflow-hidden">
                <DataTable
                    columns={columns}
                    data={apiKeys}
                    isLoading={isLoading}
                    manualPagination={true}
                    pageCount={pageCount}
                    pageIndex={pagination.pageIndex}
                    pageSize={pagination.pageSize}
                    onPaginationChange={setPagination}
                />
            </div>

            <div className="mt-4 text-xs text-muted-foreground/60 text-right">
                Security: Never share your API keys or commit them to version control.
            </div>
        </Shell>
    )
}
