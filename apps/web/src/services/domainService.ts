import api from '@/lib/api';
import type {
    Domain,
    GetDomainResponse,
    AddDomainRequest,
    AddDomainResponse,
    VerifyDomainResponse,
    ListDomainsResponse,
    DeleteDomainResponse,
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
     * Get a single domain
     */
    get: async (id: string): Promise<Domain> => {
        const response = await api.get<GetDomainResponse>(`/v1/domains/${id}`);
        return response.domain;
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
};

export default domainService;
