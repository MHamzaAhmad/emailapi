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
} from 'lucide-react'
import { useState } from 'react'
import { useDomain, useDomainRecords, useVerifyDomain } from '@/hooks'
import type { DnsRecord, RecordStatus, DomainStatus } from '@/types'

export const Route = createFileRoute('/domains/$domainId')({
    component: DomainDetailPage,
})

function getStatusIcon(status: RecordStatus | DomainStatus) {
    switch (status) {
        case 'verified':
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

function getStatusColor(status: RecordStatus | DomainStatus) {
    switch (status) {
        case 'verified':
        case 'success':
            return 'text-green-400'
        case 'pending':
            return 'text-yellow-400'
        case 'failed':
            return 'text-red-400'
        case 'temporary_failure':
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
                            <span className={`text-sm ${getStatusColor(record.status)}`}>
                                {record.status === 'verified' ? 'Verified' : record.status}
                            </span>
                        </div>
                        <p className="text-gray-400 text-sm mt-1">{record.dnsType} Record</p>
                    </div>
                </div>
            </div>

            <div className="space-y-3">
                <div>
                    <label className="block text-gray-500 text-sm mb-1">Name</label>
                    <div className="flex items-center gap-2">
                        <code className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded text-cyan-400 font-mono text-sm break-all">
                            {record.name}
                        </code>
                        <button
                            onClick={() => handleCopy(record.name)}
                            className="p-2 hover:bg-slate-700 rounded transition-colors"
                            title="Copy name"
                        >
                            {copied ? (
                                <Check className="w-4 h-4 text-green-400" />
                            ) : (
                                <Copy className="w-4 h-4 text-gray-400" />
                            )}
                        </button>
                    </div>
                </div>

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

                {record.priority !== undefined && record.priority > 0 && (
                    <div>
                        <label className="block text-gray-500 text-sm mb-1">Priority</label>
                        <div className="px-3 py-2 bg-slate-900 border border-slate-700 rounded text-white text-sm">
                            {record.priority}
                        </div>
                    </div>
                )}

                {record.instructions && (
                    <div className="mt-4 p-3 bg-blue-500/10 border border-blue-500/30 rounded-lg">
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
    const { data: domain, isLoading: domainLoading } = useDomain(domainId)
    const { data: records, isLoading: recordsLoading } = useDomainRecords(domainId)
    const verifyMutation = useVerifyDomain()

    const isLoading = domainLoading || recordsLoading

    if (isLoading) {
        return (
            <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center">
                <div className="text-gray-400">Loading domain details...</div>
            </div>
        )
    }

    if (!domain || !records) {
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
                                {domain.verifiedForSending && (
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

                {/* Status Cards */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
                    <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-4">
                        <div className="flex items-center justify-between">
                            <span className="text-gray-400 text-sm">Ready to Send</span>
                            {records.isReadyToSend ? (
                                <CheckCircle className="w-5 h-5 text-green-400" />
                            ) : (
                                <Clock className="w-5 h-5 text-yellow-400" />
                            )}
                        </div>
                        <p className="text-white font-semibold mt-2">
                            {records.isReadyToSend ? 'Yes' : 'Not Yet'}
                        </p>
                    </div>

                    <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-4">
                        <div className="flex items-center justify-between">
                            <span className="text-gray-400 text-sm">Ready to Receive</span>
                            {records.isReadyToReceive ? (
                                <CheckCircle className="w-5 h-5 text-green-400" />
                            ) : (
                                <Clock className="w-5 h-5 text-yellow-400" />
                            )}
                        </div>
                        <p className="text-white font-semibold mt-2">
                            {records.isReadyToReceive ? 'Yes' : 'Not Yet'}
                        </p>
                    </div>

                    <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-4">
                        <div className="flex items-center justify-between">
                            <span className="text-gray-400 text-sm">Fully Configured</span>
                            {records.isFullyConfigured ? (
                                <CheckCircle className="w-5 h-5 text-green-400" />
                            ) : (
                                <Clock className="w-5 h-5 text-yellow-400" />
                            )}
                        </div>
                        <p className="text-white font-semibold mt-2">
                            {records.isFullyConfigured ? 'Yes' : 'Not Yet'}
                        </p>
                    </div>
                </div>

                {/* DKIM Records */}
                {records.dkimRecords && records.dkimRecords.length > 0 && (
                    <div className="mb-8">
                        <h2 className="text-2xl font-bold text-white mb-4">
                            DKIM Records ({records.dkimRecords.length})
                        </h2>
                        <p className="text-gray-400 mb-4">
                            DKIM (DomainKeys Identified Mail) records authenticate your emails and improve
                            deliverability. Add all {records.dkimRecords.length} CNAME records to your DNS.
                        </p>
                        <div className="grid gap-4">
                            {records.dkimRecords.map((record, index) => (
                                <DnsRecordCard key={index} record={record} />
                            ))}
                        </div>
                    </div>
                )}

                {/* SPF Record */}
                {records.spfRecord && (
                    <div className="mb-8">
                        <h2 className="text-2xl font-bold text-white mb-4">SPF Record</h2>
                        <p className="text-gray-400 mb-4">
                            SPF (Sender Policy Framework) authorizes AWS SES to send emails on your behalf.
                        </p>
                        <DnsRecordCard record={records.spfRecord} />
                    </div>
                )}

                {/* DMARC Record */}
                {records.dmarcRecord && (
                    <div className="mb-8">
                        <h2 className="text-2xl font-bold text-white mb-4">DMARC Record</h2>
                        <p className="text-gray-400 mb-4">
                            DMARC (Domain-based Message Authentication) defines how recipients should handle
                            emails that fail authentication.
                        </p>
                        <DnsRecordCard record={records.dmarcRecord} />
                    </div>
                )}

                {/* MX Records */}
                {records.mxRecords && records.mxRecords.length > 0 && (
                    <div className="mb-8">
                        <h2 className="text-2xl font-bold text-white mb-4">
                            MX Records (Inbound Email)
                        </h2>
                        <p className="text-gray-400 mb-4">
                            MX (Mail Exchange) records are required to receive inbound emails.
                        </p>
                        <div className="grid gap-4">
                            {records.mxRecords.map((record, index) => (
                                <DnsRecordCard key={index} record={record} />
                            ))}
                        </div>
                    </div>
                )}

                {/* MAIL FROM Records */}
                {records.mailFromRecords && records.mailFromRecords.length > 0 && (
                    <div className="mb-8">
                        <h2 className="text-2xl font-bold text-white mb-4">
                            MAIL FROM Records
                        </h2>
                        <p className="text-gray-400 mb-4">
                            Custom MAIL FROM domain improves deliverability by controlling the Return-Path
                            header used for bounce handling.
                            {domain.mailFromDomain && (
                                <span className="block mt-2 text-cyan-400">
                                    MAIL FROM: {domain.mailFromDomain}
                                </span>
                            )}
                        </p>
                        <div className="grid gap-4">
                            {records.mailFromRecords.map((record, index) => (
                                <DnsRecordCard key={index} record={record} />
                            ))}
                        </div>
                    </div>
                )}

                {/* Help Section */}
                <div className="bg-slate-800/30 border border-slate-700 rounded-xl p-6">
                    <h2 className="text-lg font-semibold text-white mb-3">DNS Configuration Steps</h2>
                    <div className="space-y-3 text-gray-400 text-sm">
                        <p>
                            <span className="text-cyan-400 font-mono">1.</span> Log in to your domain registrar
                            (GoDaddy, Namecheap, Cloudflare, etc.)
                        </p>
                        <p>
                            <span className="text-cyan-400 font-mono">2.</span> Navigate to DNS settings for{' '}
                            <span className="text-white font-mono">{domain.domain}</span>
                        </p>
                        <p>
                            <span className="text-cyan-400 font-mono">3.</span> Add each DNS record shown above
                            with the exact name and value
                        </p>
                        <p>
                            <span className="text-cyan-400 font-mono">4.</span> DNS propagation can take up to 72
                            hours, but typically completes within 1 hour
                        </p>
                        <p>
                            <span className="text-cyan-400 font-mono">5.</span> Click "Verify Domain" above to
                            check if DNS records have propagated
                        </p>
                    </div>
                </div>
            </div>
        </div>
    )
}
