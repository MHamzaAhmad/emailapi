import api from '@/lib/api';
import {
    SendEmailRequest,
    SendEmailResponse,
    ListEmailsRequest,
    ListEmailsResponse,
    Email,
} from '@/types';

export const emailService = {
    /**
     * Send an email
     */
    sendEmail: async (data: SendEmailRequest): Promise<SendEmailResponse> => {
        return api.post<SendEmailResponse>('/v1/send', data);
    },

    /**
     * List emails
     */
    listEmails: async (params?: ListEmailsRequest): Promise<ListEmailsResponse> => {
        return api.get<ListEmailsResponse>('/v1/emails', { params });
    },

    /**
     * Get an email by ID
     */
    getEmail: async (id: string): Promise<Email> => {
        return api.get<Email>(`/v1/emails/${id}`);
    },
};

export default emailService;
