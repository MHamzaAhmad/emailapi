import { useMutation, useQuery } from '@tanstack/react-query';
import { emailService } from '@/services/emailService';
import { SendEmailRequest, ListEmailsRequest } from '@/types';

export const useSendEmail = () => {
    return useMutation({
        mutationFn: (data: SendEmailRequest) => emailService.sendEmail(data),
    });
};

export const useListEmails = (params?: ListEmailsRequest) => {
    return useQuery({
        queryKey: ['emails', params],
        queryFn: () => emailService.listEmails(params),
    });
};

export const useGetEmail = (id: string) => {
    return useQuery({
        queryKey: ['email', id],
        queryFn: () => emailService.getEmail(id),
        enabled: !!id,
    });
};
