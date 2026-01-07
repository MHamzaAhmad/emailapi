import { useMutation } from '@tanstack/react-query';
import { emailService } from '@/services/emailService';
import { SendEmailRequest } from '@/types';

export const useSendEmail = () => {
    return useMutation({
        mutationFn: (data: SendEmailRequest) => emailService.sendEmail(data),
    });
};


