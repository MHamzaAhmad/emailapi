// API Response types
export interface ApiResponse<T> {
    data: T;
    message?: string;
}

export interface ListResponse<T> {
    data: T[];
}

export interface ApiError {
    message: string;
    code?: string;
    details?: Record<string, unknown>;
}

// Pagination
export interface PaginationParams {
    limit?: number;
    offset?: number;
}
