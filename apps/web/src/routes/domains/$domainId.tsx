import { createFileRoute, Link } from '@tanstack/react-router'
import {
    Globe,
    ArrowLeft,
    Copy,
    Check,
    CheckCircle,
    Clock,
    AlertCircle,
    RefreshCw,
    Info,
    Loader2,
} from 'lucide-react'
import { useState } from 'react'
import { useDomain, useVerifyDomain } from '@/hooks'
import type { DnsRecord, RecordStatus, DomainStatus } from '@/types'

export const Route = createFileRoute('/domains/$domainId')({
    component: DomainDetailPage,
})

function getStatusIcon(status: RecordStatus | DomainStatus) {
    switch (status) {
        case 'found':
        case 'ready':
            return <CheckCircle className="w-5 h-5 text-green-400" />
        case 'verifying':
            return <Loader2 className="w-5 h-5 text-blue-400 animate-spin" />
        case 'pending':
            return <Clock className="w-5 h-5 text-yellow-400" />
        case 'failed':
        case 'missing':
        case 'mismatch':
            return <AlertCircle className="w-5 h-5 text-red-400" />
        case 'degraded':
            return <AlertCircle className="w-5 h-5 text-orange-400" />
        default:
            return <Clock className="w-5 h-5 text-gray-400" />
    }
}

function getStatusColor(status: RecordStatus | DomainStatus) {
    switch (status) {
        case 'found':
        case 'ready':
            return 'text-green-400'
        case 'verifying':
            return 'text-blue-400'
        case 'pending':
            return 'text-yellow-400'
        case 'failed':
        case 'missing':
        case 'mismatch':
            return 'text-red-400'
        case 'degraded':
            return 'text-orange-400'
        default:
            return 'text-gray-400'
    }
}

function DnsRecordCard({ record }: { record: DnsRecord }) {
    const [copied, setCopied] = useState(false)

    const handleCopy = async (text: string) => {
        await navigator.clipboard.writeText(text)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
    }

    return (
        <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
            <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                    {getStatusIcon(record.status)}
                    <div>
                        <div className="flex items-center gap-2">
                            <span className="text-white font-semibold">
                                {record.recordType.toUpperCase().replace(/_/g, ' ')}
                            </span>
                            <span className={`text-sm ${getStatusColor(record.status)} capitalize`}>
                                {record.status}
                            </span>
                        </div>
                        <p className="text-gray-400 text-sm mt-1">{record.type} Record</p>
                    </div>
                </div>
            </div>

            <div className="space-y-4">
                {/* Host/Name */}
                <div>
                    <div className="flex items-center justify-between mb-1">
                        <label className="text-gray-500 text-sm">Host</label>
                        <span className="text-xs text-gray-600 bg-slate-900 px-2 py-0.5 rounded">
                            Use this if your provider asks for the full name
                        </span>
                    </div>
                    <div className="flex items-center gap-2">
                        <code className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded text-cyan-400 font-mono text-sm break-all">
                            {record.name}
                        </code>
                        <button
                            onClick={() => handleCopy(record.name)}
                            className="p-2 hover:bg-slate-700 rounded transition-colors"
                            title="Copy full name"
                        >
                            {copied ? (
                                <Check className="w-4 h-4 text-green-400" />
                            ) : (
                                <Copy className="w-4 h-4 text-gray-400" />
                            )}
                        </button>
                    </div>
                    {record.nameShort && (
                        <div className="flex items-center gap-2 mt-2">
                            <span className="text-xs text-gray-500 w-12">Short:</span>
                            <code className="text-gray-300 font-mono text-xs select-all">
                                {record.nameShort}
                            </code>
                            <button
                                onClick={() => handleCopy(record.nameShort)}
                                className="ml-auto text-xs text-gray-500 hover:text-white"
                            >
                                Copy
                            </button>
                        </div>
                    )}
                </div>

                {/* Value */}
                <div>
                    <label className="block text-gray-500 text-sm mb-1">Value</label>
                    <div className="flex items-center gap-2">
                        <code className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded text-cyan-400 font-mono text-sm break-all">
                            {record.value}
                        </code>
                        <button
                            onClick={() => handleCopy(record.value)}
                            className="p-2 hover:bg-slate-700 rounded transition-colors"
                            title="Copy value"
                        >
                            {copied ? (
                                <Check className="w-4 h-4 text-green-400" />
                            ) : (
                                <Copy className="w-4 h-4 text-gray-400" />
                            )}
                        </button>
                    </div>
                </div>

                {/* Discovered Value (if mismatch) */}
                {record.status === 'mismatch' && record.discoveredValue && (
                    <div className="p-3 bg-red-500/10 border border-red-500/30 rounded-lg">
                        <label className="block text-red-400 text-xs mb-1 font-semibold">FOUND IN DNS (MISMATCH):</label>
                        <code className="block text-red-300 font-mono text-xs break-all">
                            {record.discoveredValue}
                        </code>
                    </div>
                )}

                {/* Priority for MX */}
                {record.priority !== undefined && record.priority > 0 && (
                    <div>
                        <label className="block text-gray-500 text-sm mb-1">Priority</label>
                        <div className="px-3 py-2 bg-slate-900 border border-slate-700 rounded text-white text-sm inline-block">
                            {record.priority}
                        </div>
                    </div>
                )}

                {/* Instructions */}
                {record.instructions && (
                    <div className="p-3 bg-blue-500/10 border border-blue-500/30 rounded-lg">
                        <div className="flex items-start gap-2">
                            <Info className="w-4 h-4 text-blue-400 mt-0.5 flex-shrink-0" />
                            <p className="text-blue-300 text-sm">{record.instructions}</p>
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}

function DomainDetailPage() {
    const { domainId } = Route.useParams()
    const { data: domain, isLoading } = useDomain(domainId)
    const verifyMutation = useVerifyDomain()

    if (isLoading) {
        return (
            <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center">
                <div className="text-gray-400">Loading domain details...</div>
            </div>
        )
    }

    if (!domain) {
        return (
            <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 p-6">
                <div className="max-w-5xl mx-auto">
                    <Link
                        to="/domains"
                        className="inline-flex items-center gap-2 text-cyan-400 hover:text-cyan-300 mb-6"
                    >
                        <ArrowLeft className="w-4 h-4" />
                        Back to Domains
                    </Link>
                    <div className="bg-red-500/10 border border-red-500/50 rounded-lg p-4 flex items-center gap-3">
                        <AlertCircle className="w-5 h-5 text-red-400" />
                        <p className="text-red-400">Domain not found</p>
                    </div>
                </div>
            </div>
        )
    }

    const { records, summary } = domain

    return (
        <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 p-6">
            <div className="max-w-5xl mx-auto">
                {/* Header */}
                <Link
                    to="/domains"
                    className="inline-flex items-center gap-2 text-cyan-400 hover:text-cyan-300 mb-6"
                >
                    <ArrowLeft className="w-4 h-4" />
                    Back to Domains
                </Link>

                <div className="flex items-center justify-between mb-8">
                    <div className="flex items-center gap-3">
                        <div className="p-3 bg-cyan-500/10 rounded-xl">
                            <Globe className="w-8 h-8 text-cyan-400" />
                        </div>
                        <div>
                            <h1 className="text-3xl font-bold text-white">{domain.domain}</h1>
                            <div className="flex items-center gap-3 mt-1">
                                <span className="text-gray-400">Region: {domain.region}</span>
                                {summary?.canSend && (
                                    <span className="flex items-center gap-1 text-green-400 text-sm">
                                        <CheckCircle className="w-4 h-4" />
                                        Ready to send
                                    </span>
                                )}
                            </div>
                        </div>
                    </div>
                    <button
                        onClick={() => verifyMutation.mutate(domainId)}
                        disabled={verifyMutation.isPending}
                        className="flex items-center gap-2 px-4 py-2 bg-cyan-500 hover:bg-cyan-600 disabled:bg-slate-600 text-white font-semibold rounded-lg transition-colors"
                    >
                        <RefreshCw
                            className={`w-4 h-4 ${verifyMutation.isPending ? 'animate-spin' : ''}`}
                        />
                        {verifyMutation.isPending ? 'Verifying...' : 'Verify Domain'}
                    </button>
                </div>

                {/* Status Summary */}
                <div className="grid grid-cols-1 gap-4 mb-8">
                    <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6 flex flex-col md:flex-row items-center justify-between gap-4">
                        <div>
                            <h3 className="text-lg font-semibold text-white mb-1">
                                {summary?.canSend ? "You're all set!" : "Verification Required"}
                            </h3>
                            <p className="text-gray-400">
                                {summary?.message}
                            </p>
                        </div>
                        {summary?.nextAction === 'CONFIGURE_DNS' && (
                            <div className="flex items-center gap-2 px-4 py-2 bg-yellow-500/10 text-yellow-500 rounded-lg border border-yellow-500/20">
                                <AlertCircle className="w-5 h-5" />
                                <span className="font-medium">Action Required: Add Records Below</span>
                            </div>
                        )}
                        {summary?.nextAction === 'WAIT' && (
                            <div className="flex items-center gap-2 px-4 py-2 bg-blue-500/10 text-blue-400 rounded-lg border border-blue-500/20">
                                <Clock className="w-5 h-5" />
                                <span className="font-medium">Waiting for Propagation</span>
                            </div>
                        )}
                    </div>
                </div>

                {/* DKIM Records */}
                {records?.dkimRecords && records.dkimRecords.length > 0 && (
                    <div className="mb-8">
                        <h2 className="text-2xl font-bold text-white mb-4">
                            DKIM Records ({records.dkimRecords.length})
                        </h2>
                        <p className="text-gray-400 mb-4">
                            DKIM (DomainKeys Identified Mail) records authenticate your emails. Add all {records.dkimRecords.length} records.
                        </p>
                        <div className="grid gap-4">
                            {records.dkimRecords.map((record, index) => (
                                <DnsRecordCard key={index} record={record} />
                            ))}
                        </div>
                    </div>
                )}

                {/* SPF Record */}
                {records?.spfRecord && (
                    <div className="mb-8">
                        <h2 className="text-2xl font-bold text-white mb-4">SPF Record</h2>
                        <p className="text-gray-400 mb-4">
                            SPF (Sender Policy Framework) authorizes us to send emails on your behalf.
                        </p>
                        <DnsRecordCard record={records.spfRecord} />
                    </div>
                )}

                {/* DMARC Record */}
                {records?.dmarcRecord && (
                    <div className="mb-8">
                        <h2 className="text-2xl font-bold text-white mb-4">DMARC Record</h2>
                        <p className="text-gray-400 mb-4">
                            DMARC defines how recipients handle authentication failures.
                        </p>
                        <DnsRecordCard record={records.dmarcRecord} />
                    </div>
                )}

                {/* MX Records */}
                {records?.mxRecords && records.mxRecords.length > 0 && (
                    <div className="mb-8">
                        <h2 className="text-2xl font-bold text-white mb-4">
                            MX Records
                        </h2>
                        <p className="text-gray-400 mb-4">
                            MX records are required to receive inbound emails.
                        </p>
                        <div className="grid gap-4">
                            {records.mxRecords.map((record, index) => (
                                <DnsRecordCard key={index} record={record} />
                            ))}
                        </div>
                    </div>
                )}

                {/* MAIL FROM Records */}
                {records?.mailFromRecords && records.mailFromRecords.length > 0 && (
                    <div className="mb-8">
                        <h2 className="text-2xl font-bold text-white mb-4">
                            MAIL FROM Records
                        </h2>
                        <p className="text-gray-400 mb-4">
                            Custom MAIL FROM improves deliverability by using your own domain for bounce handling.
                        </p>
                        <div className="grid gap-4">
                            {records.mailFromRecords.map((record, index) => (
                                <DnsRecordCard key={index} record={record} />
                            ))}
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}
