import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Add01Icon,
    MoreHorizontalIcon,
    Copy01Icon,
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
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useApiKeys, useCreateApiKey, useRevokeApiKey } from '@/hooks'
import type { ApiKey } from '@/types'

export const Route = createFileRoute('/api-keys')(
    {
        component: ApiKeysPage,
    }
)

function ApiKeysPage() {
    return (
        <AuthGuard>
            <ApiKeysContent />
        </AuthGuard>
    )
}

function ApiKeysContent() {
    const [isCreateOpen, setIsCreateOpen] = useState(false)
    const [newKeyName, setNewKeyName] = useState('')
    const [createdKey, setCreatedKey] = useState<string | null>(null)

    const { data: apiKeys, isLoading } = useApiKeys()
    const createApiKey = useCreateApiKey()
    const revokeApiKey = useRevokeApiKey()

    const handleCreate = async () => {
        if (!newKeyName.trim()) return

        try {
            const result = await createApiKey.mutateAsync({
                name: newKeyName,
                scopes: ['email:send'],
                environment: 'live',
            })
            setCreatedKey(result.rawKey)
            setNewKeyName('')
        } catch (error) {
            console.error('Failed to create API key:', error)
        }
    }

    const handleCopy = async (text: string) => {
        await navigator.clipboard.writeText(text)
    }

    const handleRevoke = async (id: string) => {
        if (!confirm('Are you sure you want to revoke this API key?')) return
        await revokeApiKey.mutateAsync(id)
    }

    return (
        <Shell>
            <PageHeader
                title="API Keys"
                description="Manage your API keys for authentication"
                actions={
                    <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
                        <DialogTrigger asChild>
                            <Button size="sm">
                                <HugeiconsIcon icon={Add01Icon} size={12} strokeWidth={1.5} />
                                Create key
                            </Button>
                        </DialogTrigger>
                        <DialogContent>
                            {createdKey ? (
                                <>
                                    <DialogHeader>
                                        <DialogTitle>API Key Created</DialogTitle>
                                        <DialogDescription>
                                            Copy your API key now. You won't see it again.
                                        </DialogDescription>
                                    </DialogHeader>
                                    <div className="space-y-3">
                                        <div className="flex items-center gap-2">
                                            <code className="flex-1 rounded-md bg-muted px-3 py-2 text-xs font-mono break-all">
                                                {createdKey}
                                            </code>
                                            <Button
                                                variant="outline"
                                                size="icon"
                                                onClick={() => handleCopy(createdKey)}
                                            >
                                                <HugeiconsIcon icon={Copy01Icon} size={14} strokeWidth={1.5} />
                                            </Button>
                                        </div>
                                    </div>
                                    <DialogFooter>
                                        <Button
                                            onClick={() => {
                                                setCreatedKey(null)
                                                setIsCreateOpen(false)
                                            }}
                                        >
                                            Done
                                        </Button>
                                    </DialogFooter>
                                </>
                            ) : (
                                <>
                                    <DialogHeader>
                                        <DialogTitle>Create API Key</DialogTitle>
                                        <DialogDescription>
                                            Give your key a name to identify it later.
                                        </DialogDescription>
                                    </DialogHeader>
                                    <div className="space-y-3">
                                        <div className="space-y-1.5">
                                            <Label htmlFor="name">Name</Label>
                                            <Input
                                                id="name"
                                                placeholder="e.g., Production Server"
                                                value={newKeyName}
                                                onChange={(e) => setNewKeyName(e.target.value)}
                                            />
                                        </div>
                                    </div>
                                    <DialogFooter>
                                        <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                                            Cancel
                                        </Button>
                                        <Button
                                            onClick={handleCreate}
                                            disabled={!newKeyName.trim() || createApiKey.isPending}
                                        >
                                            {createApiKey.isPending ? 'Creating...' : 'Create'}
                                        </Button>
                                    </DialogFooter>
                                </>
                            )}
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
                    ) : !apiKeys?.length ? (
                        <div className="p-4 text-center text-xs text-muted-foreground">
                            No API keys yet. Create one to get started.
                        </div>
                    ) : (
                        <div className="divide-y divide-border">
                            {apiKeys.map((key: ApiKey) => (
                                <div key={key.id} className="flex items-center justify-between px-3 py-2.5">
                                    <div className="flex items-center gap-3">
                                        <div>
                                            <div className="text-xs font-medium">{key.name}</div>
                                            <code className="text-2xs text-muted-foreground font-mono">
                                                {key.keyPrefix}...
                                            </code>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <Badge variant={key.isActive ? 'success' : 'secondary'}>
                                            {key.isActive ? 'active' : 'revoked'}
                                        </Badge>
                                        <DropdownMenu>
                                            <DropdownMenuTrigger asChild>
                                                <Button variant="ghost" size="icon">
                                                    <HugeiconsIcon icon={MoreHorizontalIcon} size={14} strokeWidth={1.5} />
                                                </Button>
                                            </DropdownMenuTrigger>
                                            <DropdownMenuContent align="end">
                                                <DropdownMenuItem onClick={() => handleCopy(key.keyPrefix)}>
                                                    <HugeiconsIcon icon={Copy01Icon} size={12} strokeWidth={1.5} />
                                                    Copy prefix
                                                </DropdownMenuItem>
                                                {key.isActive && (
                                                    <DropdownMenuItem
                                                        className="text-destructive"
                                                        onClick={() => handleRevoke(key.id)}
                                                    >
                                                        <HugeiconsIcon icon={Delete01Icon} size={12} strokeWidth={1.5} />
                                                        Revoke
                                                    </DropdownMenuItem>
                                                )}
                                            </DropdownMenuContent>
                                        </DropdownMenu>
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
