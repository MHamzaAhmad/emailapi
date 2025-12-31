import { adminClient } from '@/lib/connect';
import type {
    ListAdminUsersRequest,
    ListAdminUsersResponse,
    ListFlaggedUsersRequest,
    ListFlaggedUsersResponse,
    UserDetails,
    SuspendUserResponse,
    UnsuspendUserResponse,
} from '@/lib/connect';

/**
 * Admin Service
 * Handles admin-only API operations using Connect RPC
 */
export const adminService = {
    /**
     * List all users with pagination
     */
    listUsers: async (params?: Partial<Omit<ListAdminUsersRequest, '$typeName'>>): Promise<ListAdminUsersResponse> => {
        return adminClient().listUsers({
            pageSize: params?.pageSize ?? 20,
            offset: params?.offset ?? 0,
        });
    },

    /**
     * List flagged users with pagination
     */
    listFlaggedUsers: async (params?: Partial<Omit<ListFlaggedUsersRequest, '$typeName'>>): Promise<ListFlaggedUsersResponse> => {
        return adminClient().listFlaggedUsers({
            pageSize: params?.pageSize ?? 50,
            offset: params?.offset ?? 0,
        });
    },

    /**
     * Get detailed user information including reputation
     */
    getUserDetails: async (userId: string): Promise<UserDetails> => {
        return adminClient().getUserDetails({ userId });
    },

    /**
     * Suspend a user
     */
    suspendUser: async (userId: string, reason: string): Promise<SuspendUserResponse> => {
        return adminClient().suspendUser({ userId, reason });
    },

    /**
     * Unsuspend a user
     */
    unsuspendUser: async (userId: string): Promise<UnsuspendUserResponse> => {
        return adminClient().unsuspendUser({ userId });
    },
};

export default adminService;
