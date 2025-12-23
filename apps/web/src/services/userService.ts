import { userClient } from '@/lib/connect';
import type {
    User,
    CreateUserRequest,
    CreateUserResponse,
    UpdateUserRequest,
    ListUsersRequest,
    ListUsersResponse,
} from '@/lib/connect';

/**
 * User Service
 * Handles all user-related API operations using Connect RPC
 */
export const userService = {
    /**
     * List all users
     */
    list: async (params?: Partial<Omit<ListUsersRequest, '$typeName'>>): Promise<ListUsersResponse> => {
        return userClient().listUsers({
            pageSize: params?.pageSize ?? 20,
            offset: params?.offset ?? 0,
            pageToken: params?.pageToken ?? '',
        });
    },

    /**
     * Create a new user
     */
    create: async (data: Omit<CreateUserRequest, '$typeName'>): Promise<CreateUserResponse> => {
        return userClient().createUser(data);
    },

    /**
     * Get the current authenticated user
     */
    getCurrentUser: async (): Promise<User> => {
        return userClient().getCurrentUser({});
    },

    /**
     * Update a user by ID
     */
    update: async (id: string, data: Omit<UpdateUserRequest, 'id' | '$typeName'>): Promise<User> => {
        return userClient().updateUser({ id, ...data });
    },
};

export default userService;
