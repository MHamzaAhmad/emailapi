import { createFileRoute } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Key01Icon,
    Add01Icon,
    Copy01Icon,
    Delete01Icon,
    ViewIcon,
    ViewOffIcon,
} from '@hugeicons/core-free-icons'
import { Shell, PageHeader } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

export const Route = createFileRoute('/api-keys')(
    {
        component: APIKeysPage,
    }
)

function APIKeysPage() {
    return (
        <AuthGuard>
            <APIKeysContent />
        </AuthGuard>
    )
}

function APIKeysContent() {
    const [keys] = useState([
        { id: '1', name: 'Production Key', prefix: 'em_live_', created: '2025-12-20', lastUsed: '2 hours ago', environment: 'live' },
        { id: '2', name: 'Testing Key', prefix: 'em_test_', created: '2025-12-23', lastUsed: 'Never', environment: 'test' },
    ])
    const [showKeys, setShowKeys] = useState<Record<string, boolean>>({})

    const toggleShow = (id: string) => {
        setShowKeys(prev => ({ ...prev, [id]: !prev[id] }))
    }

    return (
        <Shell>
            <PageHeader
                title="API Keys"
                description="Manage your secret keys to authenticate API requests."
                actions={
                    <Button variant="default" className="gap-2">
                        <HugeiconsIcon icon={Add01Icon} size={14} />
                        <span>Create Key</span>
                    </Button>
                }
            />

            <div className="card-saas !p-0 overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                        <thead className="bg-muted/50 text-muted-foreground text-xs uppercase tracking-wider">
                            <tr>
                                <th className="px-6 py-3 font-medium">Name</th>
                                <th className="px-6 py-3 font-medium">Key Prefix</th>
                                <th className="px-6 py-3 font-medium">Environment</th>
                                <th className="px-6 py-3 font-medium">Created</th>
                                <th className="px-6 py-3 font-medium text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y">
                            {keys.map((key) => (
                                <tr key={key.id} className="hover:bg-muted/30 transition-colors">
                                    <td className="px-6 py-4 font-medium">{key.name}</td>
                                    <td className="px-6 py-4">
                                        <div className="flex items-center gap-2 font-mono text-xs text-muted-foreground">
                                            <span>{key.prefix}</span>
                                            <span>{showKeys[key.id] ? "pk_live_8f0a...92b1" : "••••••••••••••••"}</span>
                                            <button
                                                onClick={() => toggleShow(key.id)}
                                                className="ml-1 hover:text-foreground transition-colors"
                                            >
                                                <HugeiconsIcon icon={showKeys[key.id] ? ViewOffIcon : ViewIcon} size={12} />
                                            </button>
                                        </div>
                                    </td>
                                    <td className="px-6 py-4">
                                        <Badge variant={key.environment === 'live' ? 'default' : 'secondary'}>
                                            {key.environment}
                                        </Badge>
                                    </td>
                                    <td className="px-6 py-4 text-muted-foreground text-xs">{key.created}</td>
                                    <td className="px-6 py-4 text-right">
                                        <div className="flex items-center justify-end gap-1">
                                            <Button variant="ghost" size="icon" className="h-8 w-8">
                                                <HugeiconsIcon icon={Copy01Icon} size={14} />
                                            </Button>
                                            <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive">
                                                <HugeiconsIcon icon={Delete01Icon} size={14} />
                                            </Button>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </div>

            <div className="mt-8 card-saas border-dashed border-2 bg-muted/20 flex flex-col items-center text-center py-10">
                <div className="p-3 rounded-full bg-background mb-4">
                    <HugeiconsIcon icon={Key01Icon} size={24} className="text-muted-foreground" />
                </div>
                <h3 className="text-base font-semibold mb-2">Security Best Practices</h3>
                <p className="text-sm text-muted-foreground max-w-sm leading-relaxed">
                    Never commit your API keys to version control. Use environment variables to manage them securely in production.
                </p>
            </div>
        </Shell>
    )
}
