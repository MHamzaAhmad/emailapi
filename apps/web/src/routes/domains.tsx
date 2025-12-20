import { createFileRoute, Link, useParams } from '@tanstack/react-router'
import {
    Globe,
    Plus,
    Trash2,
    RefreshCw,
    CheckCircle,
    AlertCircle,
    Clock,
    Copy,
    Check,
    ChevronRight,
} from 'lucide-react'
import { useState } from 'react'
import { useDomains, useAddDomain, useDeleteDomain, useVerifyDomain } from '@/hooks'
import type { Domain, DomainStatus } from '@/types'

export const Route = createFileRoute('/domains')({
    component: DomainsPage,
})

function getStatusIcon(status: DomainStatus) {
    switch (status) {
        case 'success':
            return <CheckCircle className="w-5 h-5 text-green-400" />
        case 'pending':
            return <Clock className="w-5 h-5 text-yellow-400" />
        case 'failed':
            return <AlertCircle className="w-5 h-5 text-red-400" />
        case 'temporary_failure':
            return <AlertCircle className="w-5 h-5 text-orange-400" />
        default:
            return <Clock className="w-5 h-5 text-gray-400" />
    }
}

function getStatusLabel(status: DomainStatus) {
    switch (status) {
        case 'success':
            return 'Verified'
        case 'pending':
            return 'Pending'
        case 'failed':
            return 'Failed'
        case 'temporary_failure':
            return 'Temporary Failure'
        default:
            return status
    }
}

function DomainsPage() {
    const [isAdding, setIsAdding] = useState(false)
    const [newDomain, setNewDomain] = useState('')

    const { data: domains, isLoading, error } = useDomains()
    const addMutation = useAddDomain()
    const deleteMutation = useDeleteDomain()
    const verifyMutation = useVerifyDomain()

    const handleAdd = async () => {
        if (!newDomain.trim()) return

        try {
            await addMutation.mutateAsync({ domain: newDomain.trim() })
            setNewDomain('')
            setIsAdding(false)
        } catch (err) {
            console.error('Failed to add domain:', err)
        }
    }

    if (isLoading) {
        return (
            <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center">
                <div className="text-gray-400">Loading domains...</div>
            </div>
        )
    }

    if (error) {
        return (
            <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 p-6">
                <div className="max-w-5xl mx-auto">
                    <div className="bg-red-500/10 border border-red-500/50 rounded-lg p-4 flex items-center gap-3">
                        <AlertCircle className="w-5 h-5 text-red-400" />
                        <p className="text-red-400">Failed to load domains. Make sure you're authenticated.</p>
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
                        <Globe className="w-8 h-8 text-cyan-400" />
                        <h1 className="text-3xl font-bold text-white">Domains</h1>
                    </div>
                    <button
                        onClick={() => setIsAdding(true)}
                        className="flex items-center gap-2 px-4 py-2 bg-cyan-500 hover:bg-cyan-600 text-white font-semibold rounded-lg transition-colors"
                    >
                        <Plus className="w-4 h-4" />
                        Add Domain
                    </button>
                </div>

                {/* Add Form */}
                {isAdding && (
                    <div className="mb-6 bg-slate-800/50 border border-slate-700 rounded-xl p-6">
                        <h2 className="text-xl font-semibold text-white mb-4">Add New Domain</h2>
                        <div className="space-y-4">
                            <div>
                                <label className="block text-gray-400 mb-2">Domain Name</label>
                                <input
                                    type="text"
                                    value={newDomain}
                                    onChange={(e) => setNewDomain(e.target.value)}
                                    placeholder="example.com"
                                    className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-gray-500 focus:border-cyan-500 focus:outline-none"
                                />
                                <p className="text-gray-500 text-sm mt-2">
                                    Enter your root domain without subdomains (e.g., example.com)
                                </p>
                            </div>

                            <div className="flex gap-3">
                                <button
                                    onClick={handleAdd}
                                    disabled={addMutation.isPending || !newDomain.trim()}
                                    className="px-6 py-2 bg-cyan-500 hover:bg-cyan-600 disabled:bg-slate-600 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-colors"
                                >
                                    {addMutation.isPending ? 'Adding...' : 'Add Domain'}
                                </button>
                                <button
                                    onClick={() => {
                                        setIsAdding(false)
                                        setNewDomain('')
                                    }}
                                    className="px-6 py-2 bg-slate-700 hover:bg-slate-600 text-gray-300 font-semibold rounded-lg transition-colors"
                                >
                                    Cancel
                                </button>
                            </div>
                        </div>
                    </div>
                )}

                {/* Domains List */}
                <div className="space-y-4">
                    {domains && domains.length > 0 ? (
                        domains.map((domain) => (
                            <div
                                key={domain.id}
                                className="bg-slate-800/50 border border-slate-700 rounded-xl p-6"
                            >
                                <div className="flex items-start justify-between">
                                    <div className="flex-1">
                                        <div className="flex items-center gap-3 mb-2">
                                            <h3 className="text-lg font-semibold text-white">{domain.domain}</h3>
                                            <div className="flex items-center gap-2">
                                                {getStatusIcon(domain.status)}
                                                <span
                                                    className={`text-sm ${domain.status === 'success'
                                                            ? 'text-green-400'
                                                            : domain.status === 'pending'
                                                                ? 'text-yellow-400'
                                                                : 'text-red-400'
                                                        }`}
                                                >
                                                    {getStatusLabel(domain.status)}
                                                </span>
                                            </div>
                                        </div>

                                        <div className="flex items-center gap-4 text-sm text-gray-400">
                                            <span>Region: {domain.region}</span>
                                            {domain.verifiedForSending && (
                                                <span className="flex items-center gap-1 text-green-400">
                                                    <CheckCircle className="w-4 h-4" />
                                                    Ready to send
                                                </span>
                                            )}
                                            {domain.mailFromDomain && (
                                                <span>MAIL FROM: {domain.mailFromDomain}</span>
                                            )}
                                        </div>

                                        {domain.lastVerifiedAt && (
                                            <p className="text-gray-500 text-sm mt-2">
                                                Last verified: {new Date(domain.lastVerifiedAt).toLocaleString()}
                                            </p>
                                        )}
                                    </div>

                                    <div className="flex items-center gap-2">
                                        <Link
                                            to={`/domains/${domain.id}`}
                                            className="p-2 hover:bg-slate-700 rounded text-cyan-400 transition-colors"
                                            title="View DNS Records"
                                        >
                                            <ChevronRight className="w-5 h-5" />
                                        </Link>
                                        <button
                                            onClick={() => verifyMutation.mutate(domain.id)}
                                            disabled={verifyMutation.isPending}
                                            className="p-2 hover:bg-cyan-500/20 rounded text-cyan-400 transition-colors"
                                            title="Refresh verification"
                                        >
                                            <RefreshCw
                                                className={`w-5 h-5 ${verifyMutation.isPending ? 'animate-spin' : ''}`}
                                            />
                                        </button>
                                        <button
                                            onClick={() => {
                                                if (
                                                    confirm(
                                                        'Are you sure you want to delete this domain? This action cannot be undone.'
                                                    )
                                                ) {
                                                    deleteMutation.mutate(domain.id)
                                                }
                                            }}
                                            disabled={deleteMutation.isPending}
                                            className="p-2 hover:bg-red-500/20 rounded text-red-400 transition-colors"
                                            title="Delete"
                                        >
                                            <Trash2 className="w-5 h-5" />
                                        </button>
                                    </div>
                                </div>
                            </div>
                        ))
                    ) : (
                        <div className="text-center py-12 text-gray-400">
                            <Globe className="w-12 h-12 mx-auto mb-4 opacity-50" />
                            <p>No domains yet. Add one to get started.</p>
                        </div>
                    )}
                </div>

                {/* Help Section */}
                <div className="mt-8 bg-slate-800/30 border border-slate-700 rounded-xl p-6">
                    <h2 className="text-lg font-semibold text-white mb-3">Setting Up Your Domain</h2>
                    <div className="space-y-3 text-gray-400 text-sm">
                        <p>
                            <span className="text-cyan-400 font-mono">1.</span> Add your domain using the button above
                        </p>
                        <p>
                            <span className="text-cyan-400 font-mono">2.</span> Click on a domain to view required DNS records
                        </p>
                        <p>
                            <span className="text-cyan-400 font-mono">3.</span> Add the DNS records to your domain registrar
                        </p>
                        <p>
                            <span className="text-cyan-400 font-mono">4.</span> Click verify to check DNS propagation (can take up to 72 hours)
                        </p>
                    </div>
                </div>
            </div>
        </div>
    )
}
