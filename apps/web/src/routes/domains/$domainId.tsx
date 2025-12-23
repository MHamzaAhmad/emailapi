import { createFileRoute, Link } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    GlobeIcon,
    ArrowLeftIcon,
    CopyIcon,
    CheckmarkCircleIcon,
    ClockIcon,
    AlertCircleIcon,
    RefreshIcon,
    InformationCircleIcon,
    Loading01Icon,
} from '@hugeicons/core-free-icons'
import { useState } from 'react'
import { Shell, PageHeader } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useDomain, useVerifyDomain } from '@/hooks'
import { RecordStatus, RecordType } from '@/generated/v1/domain_pb'
import type { DnsRecord } from '@/generated/v1/domain_pb'

export const Route = createFileRoute('/domains/$domainId')({
    component: DomainDetailPage,
})

function DomainDetailPage() {
    return (
        <AuthGuard>
            <DomainDetailContent />
        </AuthGuard>
    )
}

function getRecordStatusBadge(status: RecordStatus) {
    switch (status) {
        case RecordStatus.FOUND:
            return <Badge variant="success">found</Badge>
        case RecordStatus.PENDING:
            return <Badge variant="warning">pending</Badge>
        case RecordStatus.MISSING:
            return <Badge variant="destructive">missing</Badge>
        case RecordStatus.MISMATCH:
            return <Badge variant="destructive">mismatch</Badge>
        default:
            return <Badge variant="outline">unknown</Badge>
    }
}



function getRecordTypeName(recordType: RecordType): string {
    switch (recordType) {
        case RecordType.DKIM:
            return 'DKIM'
        case RecordType.SPF:
            return 'SPF'
        case RecordType.DMARC:
            return 'DMARC'
        case RecordType.MX_INBOUND:
            return 'MX INBOUND'
        case RecordType.MAIL_FROM_MX:
            return 'MAIL FROM MX'
        case RecordType.MAIL_FROM_SPF:
            return 'MAIL FROM SPF'
        default:
            return 'UNKNOWN'
    }
}

function DnsRecordCard({ record }: { record: DnsRecord }) {
    const [copied, setCopied] = useState<string | null>(null)

    const handleCopy = async (text: string, field: string) => {
        await navigator.clipboard.writeText(text)
        setCopied(field)
        setTimeout(() => setCopied(null), 2000)
    }

    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <CardTitle className="text-xs">
                            {getRecordTypeName(record.recordType)}
                        </CardTitle>
                        {getRecordStatusBadge(record.status)}
                    </div>
                    <span className="text-2xs text-muted-foreground">{record.type} Record</span>
                </div>
            </CardHeader>
            <CardContent className="space-y-3">
                {/* Host/Name */}
                <div>
                    <div className="flex items-center justify-between mb-1">
                        <label className="text-2xs text-muted-foreground">Host</label>
                    </div>
                    <div className="flex items-center gap-2">
                        <code className="flex-1 px-2 py-1.5 bg-muted rounded text-xs font-mono break-all">
                            {record.name}
                        </code>
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleCopy(record.name, 'name')}
                        >
                            <HugeiconsIcon
                                icon={copied === 'name' ? CheckmarkCircleIcon : CopyIcon}
                                size={14}
                                strokeWidth={1.5}
                            />
                        </Button>
                    </div>
                    {record.nameShort && (
                        <div className="flex items-center gap-2 mt-1.5">
                            <span className="text-2xs text-muted-foreground">Short:</span>
                            <code className="text-2xs font-mono">{record.nameShort}</code>
                            <Button
                                variant="ghost"
                                size="sm"
                                className="ml-auto h-5 px-1.5 text-2xs"
                                onClick={() => handleCopy(record.nameShort, 'nameShort')}
                            >
                                Copy
                            </Button>
                        </div>
                    )}
                </div>

                {/* Value */}
                <div>
                    <label className="block text-2xs text-muted-foreground mb-1">Value</label>
                    <div className="flex items-center gap-2">
                        <code className="flex-1 px-2 py-1.5 bg-muted rounded text-xs font-mono break-all">
                            {record.value}
                        </code>
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleCopy(record.value, 'value')}
                        >
                            <HugeiconsIcon
                                icon={copied === 'value' ? CheckmarkCircleIcon : CopyIcon}
                                size={14}
                                strokeWidth={1.5}
                            />
                        </Button>
                    </div>
                </div>

                {/* Discovered Value (if mismatch) */}
                {record.status === RecordStatus.MISMATCH && record.discoveredValue && (
                    <div className="p-2 bg-destructive/10 border border-destructive/30 rounded">
                        <label className="block text-2xs text-destructive mb-1 font-medium">
                            FOUND IN DNS (MISMATCH):
                        </label>
                        <code className="block text-2xs font-mono break-all text-destructive">
                            {record.discoveredValue}
                        </code>
                    </div>
                )}

                {/* Priority for MX */}
                {record.priority !== undefined && record.priority > 0 && (
                    <div>
                        <label className="block text-2xs text-muted-foreground mb-1">Priority</label>
                        <span className="text-xs">{record.priority}</span>
                    </div>
                )}

                {/* Instructions */}
                {record.instructions && (
                    <div className="p-2 bg-blue-500/10 border border-blue-500/30 rounded">
                        <div className="flex items-start gap-2">
                            <HugeiconsIcon icon={InformationCircleIcon} size={14} strokeWidth={1.5} className="text-blue-400 mt-0.5" />
                            <p className="text-2xs text-blue-300">{record.instructions}</p>
                        </div>
                    </div>
                )}
            </CardContent>
        </Card>
    )
}

function DomainDetailContent() {
    const { domainId } = Route.useParams()
    const { data: domain, isLoading } = useDomain(domainId)
    const verifyMutation = useVerifyDomain()

    if (isLoading) {
        return (
            <Shell>
                <div className="flex items-center justify-center py-12">
                    <HugeiconsIcon icon={Loading01Icon} size={16} strokeWidth={1.5} className="animate-spin" />
                    <span className="ml-2 text-xs text-muted-foreground">Loading...</span>
                </div>
            </Shell>
        )
    }

    if (!domain) {
        return (
            <Shell>
                <div className="mb-4">
                    <Link to="/domains" className="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground">
                        <HugeiconsIcon icon={ArrowLeftIcon} size={14} strokeWidth={1.5} />
                        Back to Domains
                    </Link>
                </div>
                <Card>
                    <CardContent className="p-4 flex items-center gap-2 text-destructive">
                        <HugeiconsIcon icon={AlertCircleIcon} size={14} strokeWidth={1.5} />
                        <span className="text-xs">Domain not found</span>
                    </CardContent>
                </Card>
            </Shell>
        )
    }

    const { records, summary } = domain

    return (
        <Shell>
            {/* Back link */}
            <div className="mb-4">
                <Link to="/domains" className="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground">
                    <HugeiconsIcon icon={ArrowLeftIcon} size={14} strokeWidth={1.5} />
                    Back to Domains
                </Link>
            </div>

            <PageHeader
                title={domain.domain}
                description={`Region: ${domain.region}`}
                actions={
                    <Button
                        size="sm"
                        onClick={() => verifyMutation.mutate(domainId)}
                        disabled={verifyMutation.isPending}
                    >
                        <HugeiconsIcon
                            icon={RefreshIcon}
                            size={12}
                            strokeWidth={1.5}
                            className={verifyMutation.isPending ? 'animate-spin' : ''}
                        />
                        {verifyMutation.isPending ? 'Verifying...' : 'Verify'}
                    </Button>
                }
            />

            {/* Status Summary */}
            <Card className="mb-6">
                <CardContent className="p-3">
                    <div className="flex items-center justify-between">
                        <div>
                            <div className="text-xs font-medium">
                                {summary?.canSend ? "Ready to send" : "Verification required"}
                            </div>
                            <div className="text-2xs text-muted-foreground mt-0.5">
                                {summary?.message}
                            </div>
                        </div>
                        {summary?.canSend ? (
                            <Badge variant="success">
                                <HugeiconsIcon icon={CheckmarkCircleIcon} size={10} strokeWidth={1.5} />
                                Active
                            </Badge>
                        ) : summary?.nextAction === 'CONFIGURE_DNS' ? (
                            <Badge variant="warning">
                                <HugeiconsIcon icon={AlertCircleIcon} size={10} strokeWidth={1.5} />
                                Action Required
                            </Badge>
                        ) : (
                            <Badge variant="secondary">
                                <HugeiconsIcon icon={ClockIcon} size={10} strokeWidth={1.5} />
                                Waiting
                            </Badge>
                        )}
                    </div>
                </CardContent>
            </Card>

            {/* DKIM Records */}
            {records?.dkimRecords && records.dkimRecords.length > 0 && (
                <div className="mb-6">
                    <h2 className="text-sm font-medium mb-2">
                        DKIM Records ({records.dkimRecords.length})
                    </h2>
                    <p className="text-2xs text-muted-foreground mb-3">
                        Add all {records.dkimRecords.length} CNAME records to authenticate your emails.
                    </p>
                    <div className="space-y-3">
                        {records.dkimRecords.map((record, index) => (
                            <DnsRecordCard key={index} record={record} />
                        ))}
                    </div>
                </div>
            )}

            {/* SPF Record */}
            {records?.spfRecord && (
                <div className="mb-6">
                    <h2 className="text-sm font-medium mb-2">SPF Record</h2>
                    <p className="text-2xs text-muted-foreground mb-3">
                        Authorizes us to send emails on your behalf.
                    </p>
                    <DnsRecordCard record={records.spfRecord} />
                </div>
            )}

            {/* DMARC Record */}
            {records?.dmarcRecord && (
                <div className="mb-6">
                    <h2 className="text-sm font-medium mb-2">DMARC Record</h2>
                    <p className="text-2xs text-muted-foreground mb-3">
                        Defines how recipients handle authentication failures.
                    </p>
                    <DnsRecordCard record={records.dmarcRecord} />
                </div>
            )}

            {/* MX Records */}
            {records?.mxRecords && records.mxRecords.length > 0 && (
                <div className="mb-6">
                    <h2 className="text-sm font-medium mb-2">MX Records</h2>
                    <p className="text-2xs text-muted-foreground mb-3">
                        Required to receive inbound emails.
                    </p>
                    <div className="space-y-3">
                        {records.mxRecords.map((record, index) => (
                            <DnsRecordCard key={index} record={record} />
                        ))}
                    </div>
                </div>
            )}

            {/* MAIL FROM Records */}
            {records?.mailFromRecords && records.mailFromRecords.length > 0 && (
                <div className="mb-6">
                    <h2 className="text-sm font-medium mb-2">MAIL FROM Records</h2>
                    <p className="text-2xs text-muted-foreground mb-3">
                        Custom MAIL FROM for improved deliverability.
                    </p>
                    <div className="space-y-3">
                        {records.mailFromRecords.map((record, index) => (
                            <DnsRecordCard key={index} record={record} />
                        ))}
                    </div>
                </div>
            )}
        </Shell>
    )
}
