import api from '@/lib/api';
import type {
    User,
    CreateUserRequest,
    CreateUserResponse,
    UpdateUserRequest,
    ListUsersResponse,
    ListUsersRequest,
} from '@/types';

/**
 * User Service
 * Handles all user-related API operations
 */
export const userService = {
    /**
     * List all users
     */
    list: async (params?: ListUsersRequest): Promise<ListUsersResponse> => {
        return api.get<ListUsersResponse>('/v1/users', { params });
    },
    /**
     * Create a new user
     */
    create: async (data: CreateUserRequest): Promise<CreateUserResponse> => {
        return api.post<CreateUserResponse>('/v1/users', data);
    },

    /**
     * Get the current authenticated user
     */
    getCurrentUser: async (): Promise<User> => {
        return api.get<User>('/v1/users/me');
    },

    /**
     * Update a user by ID
     */
    update: async (id: string, data: UpdateUserRequest): Promise<User> => {
        return api.patch<User>(`/v1/users/${id}`, data);
    },
};

export default userService;
