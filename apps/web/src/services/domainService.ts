import api from '@/lib/api';
import type {
    Domain,
    DomainRecords,
    AddDomainRequest,
    AddDomainResponse,
    VerifyDomainResponse,
    ListDomainsResponse,
    DeleteDomainResponse,
    SetMailFromRequest,
} from '@/types';

/**
 * Domain Service
 * Handles all domain-related API operations
 */
export const domainService = {
    /**
     * Add a new domain
     */
    add: async (data: AddDomainRequest): Promise<AddDomainResponse> => {
        return api.post<AddDomainResponse>('/v1/domains', data);
    },

    /**
     * Get a domain by ID
     */
    get: async (id: string): Promise<Domain> => {
        return api.get<Domain>(`/v1/domains/${id}`);
    },

    /**
     * List all domains for the current user
     */
    list: async (): Promise<ListDomainsResponse> => {
        return api.get<ListDomainsResponse>('/v1/domains');
    },

    /**
     * Delete a domain
     */
    delete: async (id: string): Promise<DeleteDomainResponse> => {
        return api.delete<DeleteDomainResponse>(`/v1/domains/${id}`);
    },

    /**
     * Verify a domain's DNS configuration
     */
    verify: async (id: string): Promise<VerifyDomainResponse> => {
        return api.post<VerifyDomainResponse>(`/v1/domains/${id}/verify`);
    },

    /**
     * Get DNS records for a domain
     */
    getRecords: async (id: string): Promise<DomainRecords> => {
        return api.get<DomainRecords>(`/v1/domains/${id}/records`);
    },

    /**
     * Set custom MAIL FROM domain
     */
    setMailFrom: async (id: string, data: SetMailFromRequest): Promise<Domain> => {
        return api.post<Domain>(`/v1/domains/${id}/mail-from`, data);
    },
};

export default domainService;
