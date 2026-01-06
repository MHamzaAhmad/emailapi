import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adminService } from '@/services';
import { queryKeys } from '@/lib/queryClient';
import type { ListAdminUsersRequest, ListFlaggedUsersRequest } from '@/lib/connect';

/**
 * Hook to list all users (admin only)
 */
export const useAdminUsers = (params?: Partial<ListAdminUsersRequest>) => {
    return useQuery({
        queryKey: queryKeys.admin.users(params),
        queryFn: () => adminService.listUsers(params),
    });
};

/**
 * Hook to list flagged users (admin only)
 */
export const useFlaggedUsers = (params?: Partial<ListFlaggedUsersRequest>) => {
    return useQuery({
        queryKey: queryKeys.admin.flaggedUsers(params),
        queryFn: () => adminService.listFlaggedUsers(params),
    });
};

/**
 * Hook to get user details with reputation (admin only)
 */
export const useUserDetails = (userId: string) => {
    return useQuery({
        queryKey: queryKeys.admin.userDetails(userId),
        queryFn: () => adminService.getUserDetails(userId),
        enabled: !!userId,
    });
};

/**
 * Hook to suspend a user (admin only)
 */
export const useSuspendUser = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ userId, reason }: { userId: string; reason: string }) =>
            adminService.suspendUser(userId, reason),
        onSuccess: (_, { userId }) => {
            // Invalidate admin queries
            queryClient.invalidateQueries({ queryKey: queryKeys.admin.all });
            queryClient.invalidateQueries({ queryKey: queryKeys.admin.userDetails(userId) });
        },
    });
};

/**
 * Hook to unsuspend a user (admin only)
 */
export const useUnsuspendUser = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (userId: string) => adminService.unsuspendUser(userId),
        onSuccess: (_, userId) => {
            // Invalidate admin queries
            queryClient.invalidateQueries({ queryKey: queryKeys.admin.all });
            queryClient.invalidateQueries({ queryKey: queryKeys.admin.userDetails(userId) });
        },
    });
};
