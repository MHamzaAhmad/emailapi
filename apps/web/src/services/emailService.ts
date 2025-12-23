import { emailClient } from '@/lib/connect';
import type { SendEmailRequest, SendEmailResponse } from '@/lib/connect';

/**
 * Email Service
 * Handles email sending operations using Connect RPC
 */
export const emailService = {
    /**
     * Send an email
     */
    sendEmail: async (data: Omit<SendEmailRequest, '$typeName'>): Promise<SendEmailResponse> => {
        return emailClient().sendEmail(data);
    },
};

export default emailService;
