// User types matching the backend API

export type UserRole = 'admin' | 'member';

export interface User {
    id: string;
    email: string;
    name: string;
    role: UserRole;
    isActive: boolean;
    createdAt: string;
    updatedAt: string;
}

// Request DTOs
export interface CreateUserRequest {
    email: string;
    name: string;
    role?: UserRole;
}

export interface UpdateUserRequest {
    email?: string;
    name?: string;
    role?: UserRole;
    isActive?: boolean;
}

// Response DTOs
export interface CreateUserResponse {
    user: User;
    message: string;
}
