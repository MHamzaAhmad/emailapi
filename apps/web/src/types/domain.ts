// Domain types matching the backend API

export type DomainStatus = 'pending' | 'success' | 'failed' | 'temporary_failure';
export type RecordStatus = 'pending' | 'verified' | 'failed';
export type RecordType =
    | 'dkim'
    | 'spf'
    | 'dmarc'
    | 'mx_inbound'
    | 'mail_from_mx'
    | 'mail_from_spf';

export interface Domain {
    id: string;
    domain: string;
    status: DomainStatus;
    verifiedForSending: boolean;
    mailFromDomain?: string;
    mailFromStatus?: DomainStatus;
    region: string;
    createdAt: string;
    updatedAt: string;
    lastVerifiedAt?: string;
}

export interface DnsRecord {
    dnsType: string;
    name: string;
    value: string;
    priority?: number;
    recordType: RecordType;
    status: RecordStatus;
    instructions: string;
}

export interface DomainRecords {
    domain: string;
    dkimRecords: DnsRecord[];
    spfRecord?: DnsRecord;
    dmarcRecord?: DnsRecord;
    mxRecords: DnsRecord[];
    mailFromRecords: DnsRecord[];
    isReadyToSend: boolean;
    isReadyToReceive: boolean;
    isFullyConfigured: boolean;
}

// Request DTOs
export interface AddDomainRequest {
    domain: string;
}

export interface SetMailFromRequest {
    mailFromSubdomain: string;
}

// Response DTOs
export interface AddDomainResponse {
    domain: Domain;
    records: DomainRecords;
    message: string;
}

export interface VerifyDomainResponse {
    domain: Domain;
    wasRefreshed: boolean;
    nextRetryAt?: string;
    message: string;
}

export interface ListDomainsResponse {
    data: Domain[];
}

export interface DeleteDomainResponse {
    message: string;
}
