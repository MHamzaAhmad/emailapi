import { createFileRoute } from '@tanstack/react-router'
import { Key, Plus, Trash2, X, Copy, Check, AlertCircle } from 'lucide-react'
import { useState } from 'react'
import { useApiKeys, useCreateApiKey, useRevokeApiKey, useDeleteApiKey } from '@/hooks'
import type { Scope, Environment, CreateApiKeyRequest } from '@/types'

export const Route = createFileRoute('/api-keys')({
    component: ApiKeysPage,
})

const AVAILABLE_SCOPES: { value: Scope; label: string }[] = [
    { value: 'email:send', label: 'Send Emails' },
    { value: 'email:read', label: 'Read Emails' },
    { value: 'domain:read', label: 'Read Domains' },
    { value: 'domain:write', label: 'Write Domains' },
    { value: 'apikey:read', label: 'Read API Keys' },
    { value: 'apikey:write', label: 'Write API Keys' },
    { value: 'user:read', label: 'Read User' },
    { value: 'user:write', label: 'Write User' },
]

function ApiKeysPage() {
    const [isCreating, setIsCreating] = useState(false)
    const [newKeyData, setNewKeyData] = useState<CreateApiKeyRequest>({
        name: '',
        scopes: [],
        environment: 'live',
    })
    const [createdRawKey, setCreatedRawKey] = useState<string | null>(null)
    const [copiedKey, setCopiedKey] = useState(false)

    const { data: apiKeys, isLoading, error } = useApiKeys()
    const createMutation = useCreateApiKey()
    const revokeMutation = useRevokeApiKey()
    const deleteMutation = useDeleteApiKey()

    const handleCreate = async () => {
        if (!newKeyData.name || newKeyData.scopes.length === 0) return

        try {
            const result = await createMutation.mutateAsync(newKeyData)
            setCreatedRawKey(result.rawKey)
            setNewKeyData({ name: '', scopes: [], environment: 'live' })
        } catch (err) {
            console.error('Failed to create API key:', err)
        }
    }

    const handleCopyKey = async () => {
        if (!createdRawKey) return
        await navigator.clipboard.writeText(createdRawKey)
        setCopiedKey(true)
        setTimeout(() => setCopiedKey(false), 2000)
    }

    const toggleScope = (scope: Scope) => {
        setNewKeyData((prev) => ({
            ...prev,
            scopes: prev.scopes.includes(scope)
                ? prev.scopes.filter((s) => s !== scope)
                : [...prev.scopes, scope],
        }))
    }

    if (isLoading) {
        return (
            <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center">
                <div className="text-gray-400">Loading API keys...</div>
            </div>
        )
    }

    if (error) {
        return (
            <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 p-6">
                <div className="max-w-5xl mx-auto">
                    <div className="bg-red-500/10 border border-red-500/50 rounded-lg p-4 flex items-center gap-3">
                        <AlertCircle className="w-5 h-5 text-red-400" />
                        <p className="text-red-400">Failed to load API keys. Make sure you're authenticated.</p>
                    </div>
                </div>
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 p-6">
            <div className="max-w-5xl mx-auto">
                {/* Header */}
                <div className="flex items-center justify-between mb-8">
                    <div className="flex items-center gap-3">
                        <Key className="w-8 h-8 text-cyan-400" />
                        <h1 className="text-3xl font-bold text-white">API Keys</h1>
                    </div>
                    <button
                        onClick={() => setIsCreating(true)}
                        className="flex items-center gap-2 px-4 py-2 bg-cyan-500 hover:bg-cyan-600 text-white font-semibold rounded-lg transition-colors"
                    >
                        <Plus className="w-4 h-4" />
                        Create API Key
                    </button>
                </div>

                {/* Created Key Alert */}
                {createdRawKey && (
                    <div className="mb-6 bg-green-500/10 border border-green-500/50 rounded-lg p-4">
                        <div className="flex items-start justify-between">
                            <div>
                                <h3 className="text-green-400 font-semibold mb-2">API Key Created!</h3>
                                <p className="text-gray-400 text-sm mb-3">
                                    Copy this key now. You won't be able to see it again.
                                </p>
                                <div className="flex items-center gap-2">
                                    <code className="px-3 py-2 bg-slate-800 rounded text-cyan-400 font-mono text-sm">
                                        {createdRawKey}
                                    </code>
                                    <button
                                        onClick={handleCopyKey}
                                        className="p-2 hover:bg-slate-700 rounded transition-colors"
                                    >
                                        {copiedKey ? (
                                            <Check className="w-4 h-4 text-green-400" />
                                        ) : (
                                            <Copy className="w-4 h-4 text-gray-400" />
                                        )}
                                    </button>
                                </div>
                            </div>
                            <button
                                onClick={() => setCreatedRawKey(null)}
                                className="p-1 hover:bg-slate-700 rounded"
                            >
                                <X className="w-4 h-4 text-gray-400" />
                            </button>
                        </div>
                    </div>
                )}

                {/* Create Form */}
                {isCreating && (
                    <div className="mb-6 bg-slate-800/50 border border-slate-700 rounded-xl p-6">
                        <h2 className="text-xl font-semibold text-white mb-4">Create New API Key</h2>
                        <div className="space-y-4">
                            <div>
                                <label className="block text-gray-400 mb-2">Name</label>
                                <input
                                    type="text"
                                    value={newKeyData.name}
                                    onChange={(e) =>
                                        setNewKeyData((prev) => ({ ...prev, name: e.target.value }))
                                    }
                                    placeholder="My API Key"
                                    className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-gray-500 focus:border-cyan-500 focus:outline-none"
                                />
                            </div>

                            <div>
                                <label className="block text-gray-400 mb-2">Environment</label>
                                <div className="flex gap-4">
                                    {(['live', 'dev'] as Environment[]).map((env) => (
                                        <button
                                            key={env}
                                            onClick={() =>
                                                setNewKeyData((prev) => ({ ...prev, environment: env }))
                                            }
                                            className={`px-4 py-2 rounded-lg font-medium transition-colors ${newKeyData.environment === env
                                                    ? 'bg-cyan-500 text-white'
                                                    : 'bg-slate-700 text-gray-400 hover:bg-slate-600'
                                                }`}
                                        >
                                            {env.charAt(0).toUpperCase() + env.slice(1)}
                                        </button>
                                    ))}
                                </div>
                            </div>

                            <div>
                                <label className="block text-gray-400 mb-2">Scopes</label>
                                <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                                    {AVAILABLE_SCOPES.map((scope) => (
                                        <button
                                            key={scope.value}
                                            onClick={() => toggleScope(scope.value)}
                                            className={`px-3 py-2 rounded-lg text-sm font-medium transition-colors ${newKeyData.scopes.includes(scope.value)
                                                    ? 'bg-cyan-500 text-white'
                                                    : 'bg-slate-700 text-gray-400 hover:bg-slate-600'
                                                }`}
                                        >
                                            {scope.label}
                                        </button>
                                    ))}
                                </div>
                            </div>

                            <div className="flex gap-3 pt-4">
                                <button
                                    onClick={handleCreate}
                                    disabled={createMutation.isPending || !newKeyData.name || newKeyData.scopes.length === 0}
                                    className="px-6 py-2 bg-cyan-500 hover:bg-cyan-600 disabled:bg-slate-600 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-colors"
                                >
                                    {createMutation.isPending ? 'Creating...' : 'Create Key'}
                                </button>
                                <button
                                    onClick={() => setIsCreating(false)}
                                    className="px-6 py-2 bg-slate-700 hover:bg-slate-600 text-gray-300 font-semibold rounded-lg transition-colors"
                                >
                                    Cancel
                                </button>
                            </div>
                        </div>
                    </div>
                )}

                {/* API Keys List */}
                <div className="space-y-4">
                    {apiKeys && apiKeys.length > 0 ? (
                        apiKeys.map((key) => (
                            <div
                                key={key.id}
                                className={`bg-slate-800/50 border rounded-xl p-6 ${key.isActive ? 'border-slate-700' : 'border-red-500/30 opacity-60'
                                    }`}
                            >
                                <div className="flex items-start justify-between">
                                    <div>
                                        <div className="flex items-center gap-3 mb-2">
                                            <h3 className="text-lg font-semibold text-white">{key.name}</h3>
                                            <span
                                                className={`px-2 py-0.5 rounded text-xs font-medium ${key.environment === 'live'
                                                        ? 'bg-green-500/20 text-green-400'
                                                        : 'bg-yellow-500/20 text-yellow-400'
                                                    }`}
                                            >
                                                {key.environment}
                                            </span>
                                            {!key.isActive && (
                                                <span className="px-2 py-0.5 rounded text-xs font-medium bg-red-500/20 text-red-400">
                                                    Revoked
                                                </span>
                                            )}
                                        </div>
                                        <p className="text-gray-500 font-mono text-sm mb-3">{key.keyPrefix}...</p>
                                        <div className="flex flex-wrap gap-2">
                                            {key.scopes.map((scope) => (
                                                <span
                                                    key={scope}
                                                    className="px-2 py-1 bg-slate-700 rounded text-xs text-gray-400"
                                                >
                                                    {scope}
                                                </span>
                                            ))}
                                        </div>
                                        {key.lastUsedAt && (
                                            <p className="text-gray-500 text-sm mt-3">
                                                Last used: {new Date(key.lastUsedAt).toLocaleDateString()}
                                            </p>
                                        )}
                                    </div>
                                    <div className="flex gap-2">
                                        {key.isActive && (
                                            <button
                                                onClick={() => revokeMutation.mutate(key.id)}
                                                disabled={revokeMutation.isPending}
                                                className="p-2 hover:bg-yellow-500/20 rounded text-yellow-400 transition-colors"
                                                title="Revoke"
                                            >
                                                <X className="w-4 h-4" />
                                            </button>
                                        )}
                                        <button
                                            onClick={() => {
                                                if (confirm('Are you sure you want to delete this API key?')) {
                                                    deleteMutation.mutate(key.id)
                                                }
                                            }}
                                            disabled={deleteMutation.isPending}
                                            className="p-2 hover:bg-red-500/20 rounded text-red-400 transition-colors"
                                            title="Delete"
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </button>
                                    </div>
                                </div>
                            </div>
                        ))
                    ) : (
                        <div className="text-center py-12 text-gray-400">
                            <Key className="w-12 h-12 mx-auto mb-4 opacity-50" />
                            <p>No API keys yet. Create one to get started.</p>
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}
