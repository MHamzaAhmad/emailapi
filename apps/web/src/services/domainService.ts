import { domainClient } from '@/lib/connect';
import type {
    Domain,
    AddDomainResponse,
    ListDomainsResponse,
    DeleteDomainResponse,
    VerifyDomainResponse,
} from '@/lib/connect';

/**
 * Domain Service
 * Handles all domain-related API operations using Connect RPC
 */
export const domainService = {
    /**
     * Add a new domain
     */
    add: async (data: { domain: string }): Promise<AddDomainResponse> => {
        return domainClient().addDomain(data);
    },

    /**
     * Get a single domain
     */
    get: async (id: string): Promise<Domain> => {
        const response = await domainClient().getDomain({ id });
        if (!response.domain) {
            throw new Error('Domain not found');
        }
        return response.domain;
    },

    /**
     * List all domains for the current user
     */
    list: async (params?: { page?: number; pageSize?: number }): Promise<ListDomainsResponse> => {
        return domainClient().listDomains({
            page: params?.page || 1,
            pageSize: params?.pageSize || 10,
        });
    },

    /**
     * Delete a domain
     */
    delete: async (id: string): Promise<DeleteDomainResponse> => {
        return domainClient().deleteDomain({ id });
    },

    /**
     * Verify a domain's DNS configuration
     */
    verify: async (id: string): Promise<VerifyDomainResponse> => {
        return domainClient().verifyDomain({ id });
    },
};

export default domainService;
