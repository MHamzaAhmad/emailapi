// Domain types matching the backend API

export type DomainStatus =
    | 'pending'     // DOMAIN_STATUS_PENDING
    | 'verifying'   // DOMAIN_STATUS_VERIFYING
    | 'ready'       // DOMAIN_STATUS_READY
    | 'degraded'    // DOMAIN_STATUS_DEGRADED
    | 'failed';     // DOMAIN_STATUS_FAILED

export type RecordStatus =
    | 'pending'     // RECORD_STATUS_PENDING
    | 'found'       // RECORD_STATUS_FOUND
    | 'mismatch'    // RECORD_STATUS_MISMATCH
    | 'missing';    // RECORD_STATUS_MISSING

export type RecordType =
    | 'dkim'
    | 'spf'
    | 'dmarc'
    | 'mx_inbound'
    | 'mail_from_mx'
    | 'mail_from_spf';

export interface DomainSummary {
    message: string;
    nextAction: string; // 'CONFIGURE_DNS' | 'WAIT' | 'NONE'
    recordsPending: number;
    recordsConfigured: number;
    canSend: boolean;
    canReceive: boolean;
}

export interface DnsRecord {
    type: string;        // 'CNAME', 'TXT', 'MX'
    name: string;        // Full name
    value: string;       // Expected value
    priority?: number;   // For MX
    recordType: RecordType;
    status: RecordStatus;
    nameShort: string;       // For providers wanting just subdomain
    discoveredValue: string; // What we found in DNS
    instructions: string;
}

export interface DomainRecords {
    dkimRecords: DnsRecord[];
    spfRecord?: DnsRecord;
    dmarcRecord?: DnsRecord;
    mxRecords: DnsRecord[];
    mailFromRecords: DnsRecord[];
}

export interface Domain {
    id: string;
    domain: string;
    status: DomainStatus;
    region: string;
    createdAt: string;
    updatedAt: string;
    lastCheckedAt?: string;
    summary?: DomainSummary;
    records?: DomainRecords;
}

// Request DTOs
export interface AddDomainRequest {
    domain: string;
}

// Response DTOs
export interface AddDomainResponse {
    domain: Domain;
    message: string;
}

export interface VerifyDomainResponse {
    domain: Domain;
    wasRefreshed: boolean;
    nextRetryAt?: string;
    message: string;
}

export interface GetDomainResponse {
    domain: Domain;
}

export interface ListDomainsResponse {
    data: Domain[];
}

export interface DeleteDomainResponse {
    message: string;
}
