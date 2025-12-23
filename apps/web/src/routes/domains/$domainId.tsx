import { createFileRoute, Link } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    ArrowLeft02Icon,
    Copy01Icon,
    CheckmarkCircle01Icon,
    Clock01Icon,
    AlertCircleIcon,
    RefreshIcon,
    Loading01Icon,
} from '@hugeicons/core-free-icons'
import { useState, useMemo } from 'react'
import { ColumnDef } from '@tanstack/react-table'
import { Shell } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { DataTable } from '@/components/ui/data-table'
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
    const className = "text-[10px] px-2 py-0.5 uppercase tracking-wider font-medium"
    switch (status) {
        case RecordStatus.FOUND:
            return <Badge variant="success" className={className}>Found</Badge>
        case RecordStatus.PENDING:
            return <Badge variant="warning" className={className}>Pending</Badge>
        case RecordStatus.MISSING:
            return <Badge variant="destructive" className={className}>Missing</Badge>
        case RecordStatus.MISMATCH:
            return <Badge variant="destructive" className={className}>Mismatch</Badge>
        default:
            return <Badge variant="outline" className={className}>Unknown</Badge>
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
            return 'MX Inbound'
        case RecordType.MAIL_FROM_MX:
            return 'Mail From MX'
        case RecordType.MAIL_FROM_SPF:
            return 'Mail From SPF'
        default:
            return 'Unknown'
    }
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
                icon={copied ? CheckmarkCircle01Icon : Copy01Icon}
                size={12}
                strokeWidth={1.5}
            />
        </Button>
    )
}

function DomainDetailContent() {
    const { domainId } = Route.useParams()
    const { data: domain, isLoading } = useDomain(domainId)
    const verifyMutation = useVerifyDomain()

    const records = useMemo(() => {
        if (!domain?.records) return []
        const { dkimRecords, spfRecord, dmarcRecord, mxRecords, mailFromRecords } = domain.records
        const list: DnsRecord[] = []
        if (dkimRecords) list.push(...dkimRecords)
        if (spfRecord) list.push(spfRecord)
        if (dmarcRecord) list.push(dmarcRecord)
        if (mxRecords) list.push(...mxRecords)
        if (mailFromRecords) list.push(...mailFromRecords)
        return list
    }, [domain])

    const columns = useMemo<ColumnDef<DnsRecord>[]>(
        () => [
            {
                accessorKey: 'recordType',
                header: 'Type',
                cell: ({ row }) => (
                    <div className="font-medium text-xs whitespace-nowrap">
                        {getRecordTypeName(row.original.recordType)}
                    </div>
                ),
            },
            {
                accessorKey: 'name',
                header: 'Host',
                cell: ({ row }) => (
                    <div className="flex items-center group max-w-[200px] sm:max-w-[300px]">
                        <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono truncate">
                            {row.original.name}
                        </code>
                        <div className="opacity-0 group-hover:opacity-100 transition-opacity">
                            <CopyButton text={row.original.name} />
                        </div>
                    </div>
                ),
            },
            {
                accessorKey: 'value',
                header: 'Value',
                cell: ({ row }) => (
                    <div className="flex flex-col gap-1 min-w-0 max-w-[300px] sm:max-w-[400px]">
                        <div className="flex items-center group">
                            <code className="bg-muted px-1.5 py-0.5 rounded text-xs font-mono truncate block flex-1">
                                {row.original.value}
                            </code>
                            <div className="opacity-0 group-hover:opacity-100 transition-opacity">
                                <CopyButton text={row.original.value} />
                            </div>
                        </div>
                        {row.original.status === RecordStatus.MISMATCH && row.original.discoveredValue && (
                            <div className="text-2xs text-destructive flex items-center gap-1">
                                <HugeiconsIcon icon={AlertCircleIcon} size={10} />
                                <span className="font-mono truncate">Found: {row.original.discoveredValue}</span>
                            </div>
                        )}
                    </div>
                ),
            },
            {
                accessorKey: 'status',
                header: 'Status',
                cell: ({ row }) => getRecordStatusBadge(row.original.status),
            },
        ],
        []
    )

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
                        <HugeiconsIcon icon={ArrowLeft02Icon} size={14} strokeWidth={1.5} />
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

    const { summary } = domain

    return (
        <Shell>
            {/* Header Actions */}
            <div className="flex flex-col gap-6 mb-8 mt-2">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <Link to="/domains" className="flex items-center justify-center h-8 w-8 rounded-md border border-border/40 bg-background hover:bg-muted/50 transition-colors text-muted-foreground hover:text-foreground">
                            <HugeiconsIcon icon={ArrowLeft02Icon} size={16} strokeWidth={1.5} />
                        </Link>
                        <div>
                            <div className="text-sm font-medium leading-none">{domain.domain}</div>
                            <div className="text-2xs text-muted-foreground mt-1">Region: {domain.region}</div>
                        </div>
                    </div>
                    <div className="flex items-center gap-2">
                        <Button
                            size="sm"
                            onClick={() => verifyMutation.mutate(domainId)}
                            disabled={verifyMutation.isPending}
                            className="h-8 shadow-sm"
                        >
                            <HugeiconsIcon
                                icon={RefreshIcon}
                                size={12}
                                strokeWidth={1.5}
                                className={verifyMutation.isPending ? 'animate-spin' : ''}
                            />
                            {verifyMutation.isPending ? 'Verifying...' : 'Verify'}
                        </Button>
                    </div>
                </div>
            </div>

            {/* Status Summary */}
            <Card className="mb-6">
                <CardContent className="p-3">
                    <div className="flex items-center justify-between">
                        <div>
                            <div className="text-xs font-medium">
                                {summary?.canSend ? "Ready to send" : "Verification required"}
                            </div>
                            <div className="text-2xs text-muted-foreground mt-0.5">
                                {summary?.message || "Please configure the DNS records below to verify your domain."}
                            </div>
                        </div>
                        {summary?.canSend ? (
                            <Badge variant="success" className="h-6">
                                <HugeiconsIcon icon={CheckmarkCircle01Icon} size={12} strokeWidth={1.5} className="mr-1" />
                                Active
                            </Badge>
                        ) : summary?.nextAction === 'CONFIGURE_DNS' ? (
                            <Badge variant="warning" className="h-6">
                                <HugeiconsIcon icon={AlertCircleIcon} size={12} strokeWidth={1.5} className="mr-1" />
                                Action Required
                            </Badge>
                        ) : (
                            <Badge variant="secondary" className="h-6">
                                <HugeiconsIcon icon={Clock01Icon} size={12} strokeWidth={1.5} className="mr-1" />
                                Waiting
                            </Badge>
                        )}
                    </div>
                </CardContent>
            </Card>

            <div className="space-y-4">
                <div className="flex items-center justify-between">
                    <h2 className="text-sm font-semibold">DNS Records</h2>
                </div>
                <div className="rounded-xl border border-border/40 bg-card shadow-sm overflow-hidden">
                    <DataTable
                        columns={columns}
                        data={records}
                        isLoading={isLoading}
                    />
                </div>
            </div>
        </Shell>
    )
}
