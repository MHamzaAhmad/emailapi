import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { userService } from '@/services';
import { queryKeys } from '@/lib/queryClient';
import type { UpdateUserRequest, User } from '@/types';

/**
 * Hook to get the current authenticated user
 */
export const useCurrentUser = () => {
    return useQuery({
        queryKey: queryKeys.users.me(),
        queryFn: userService.getCurrentUser,
    });
};

/**
 * Hook to create a new user
 */
export const useCreateUser = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: userService.create,
        onSuccess: (response) => {
            // Invalidate and refetch user queries
            queryClient.invalidateQueries({ queryKey: queryKeys.users.all });
            // Optionally set the current user if this is a self-registration
            queryClient.setQueryData<User>(queryKeys.users.me(), response.user);
        },
    });
};

/**
 * Hook to update a user
 */
export const useUpdateUser = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateUserRequest }) =>
            userService.update(id, data),
        onSuccess: (updatedUser, variables) => {
            // Update the cache with the new user data
            queryClient.setQueryData<User>(
                queryKeys.users.detail(variables.id),
                updatedUser
            );
            // If updating current user, update that cache too
            queryClient.setQueryData<User>(queryKeys.users.me(), (oldData) => {
                if (oldData?.id === variables.id) {
                    return updatedUser;
                }
                return oldData;
            });
        },
    });
};
