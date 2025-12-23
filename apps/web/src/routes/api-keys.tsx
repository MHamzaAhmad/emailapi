import { createFileRoute } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Add01Icon,
    Copy01Icon,
    Delete01Icon,
    ViewIcon,
    ViewOffIcon,
    AlertCircleIcon,
    Key01Icon
} from '@hugeicons/core-free-icons'
import { Shell } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { useState, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { DataTable } from '@/components/ui/data-table'
import { ColumnDef } from '@tanstack/react-table'
import { useAPIKeys, useDeleteAPIKey, useRevokeAPIKey } from '@/hooks'
import { ApiKey, Environment } from '@/generated/v1/apikey_pb'
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
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { Card, CardContent } from '@/components/ui/card'

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
                icon={copied ? ViewIcon : Copy01Icon} // Using ViewIcon as checkmark proxy or need Check icon import
                size={12}
                strokeWidth={1.5}
                className={copied ? "text-emerald-500" : ""}
            />
        </Button>
    )
}

function APIKeysContent() {
    const [pagination, setPagination] = useState({
        pageIndex: 0,
        pageSize: 10,
    })

    // Create Key Dialog State
    const [isCreateOpen, setIsCreateOpen] = useState(false)
    const [newKeyName, setNewKeyName] = useState('')
    const [newKeyEnv, setNewKeyEnv] = useState<Environment>(Environment.LIVE)
    const [createdKey, setCreatedKey] = useState<{ key: ApiKey, raw: string } | null>(null)

    const { data: apiKeysResponse, isLoading } = useAPIKeys(pagination.pageIndex + 1, pagination.pageSize)
    const apiKeys = apiKeysResponse?.data || []
    const total = apiKeysResponse?.total || 0
    const pageCount = Math.ceil(total / pagination.pageSize)

    const createMutation = useCreateAPIKey()
    const deleteMutation = useDeleteAPIKey()
    const revokeMutation = useRevokeAPIKey()
    const [showKeys, setShowKeys] = useState<Record<string, boolean>>({})

    const toggleShow = (id: string) => {
        setShowKeys(prev => ({ ...prev, [id]: !prev[id] }))
    }

    const handleCreate = async () => {
        if (!newKeyName.trim()) return
        try {
            const response = await createMutation.mutateAsync({
                name: newKeyName,
                environment: newKeyEnv,
                scopes: [], // Default scopes handled by backend
            })
            setCreatedKey({ key: response.apiKey!, raw: response.rawKey })
            setNewKeyName('')
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
                            {row.original.keyPrefix}...
                        </span>
                        {/* We don't have the full key to show/hide here anymore, only prefix is stored/returned for security usually? 
                            Actually proto has key_prefix. Real key is only shown on creation. 
                            So view/hide logic might not apply to list unless we store masked version? 
                            The original code had view/hide, but backend usually doesn't return full key.
                            We'll just show prefix.
                        */}
                    </div>
                ),
            },
            {
                accessorKey: 'environment',
                header: 'Environment',
                cell: ({ row }) => (
                    <Badge variant={row.original.environment === Environment.LIVE ? 'default' : 'secondary'} className="text-[10px] px-2 py-0 uppercase tracking-wider font-semibold">
                        {row.original.environment === Environment.LIVE ? 'Live' : 'Test'}
                    </Badge>
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
                                            <div className="space-y-1.5">
                                                <Label htmlFor="env">Environment</Label>
                                                <Select
                                                    value={String(newKeyEnv)}
                                                    onValueChange={(v) => setNewKeyEnv(Number(v))}
                                                >
                                                    <SelectTrigger>
                                                        <SelectValue />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        <SelectItem value={String(Environment.LIVE)}>Live</SelectItem>
                                                        <SelectItem value={String(Environment.DEV)}>Test</SelectItem>
                                                    </SelectContent>
                                                </Select>
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
