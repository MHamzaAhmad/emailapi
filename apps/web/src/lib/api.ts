import axios, { AxiosError, AxiosInstance, AxiosRequestConfig } from 'axios';
import type { ApiError } from '@/types';

// API base URL - defaults to /api for same-origin requests
const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';

// Create axios instance with default config
const axiosInstance: AxiosInstance = axios.create({
    baseURL: API_BASE_URL,
    headers: {
        'Content-Type': 'application/json',
    },
    timeout: 30000, // 30 seconds
});

// Request interceptor for adding auth token
axiosInstance.interceptors.request.use(
    (config) => {
        // Get token from localStorage or wherever you store it
        const token = typeof window !== 'undefined' ? localStorage.getItem('api_key') : null;
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => Promise.reject(error)
);

// Response interceptor for error handling
axiosInstance.interceptors.response.use(
    (response) => response,
    (error: AxiosError<ApiError>) => {
        // Handle different error scenarios
        if (error.response) {
            // Server responded with error status
            const apiError: ApiError = {
                message: error.response.data?.message || 'An error occurred',
                code: error.response.data?.code,
                details: error.response.data?.details,
            };
            return Promise.reject(apiError);
        } else if (error.request) {
            // Request made but no response received
            return Promise.reject({
                message: 'Network error. Please check your connection.',
                code: 'NETWORK_ERROR',
            } as ApiError);
        } else {
            // Error in request configuration
            return Promise.reject({
                message: error.message,
                code: 'REQUEST_ERROR',
            } as ApiError);
        }
    }
);

// Generic API request functions
export const api = {
    get: async <T>(url: string, config?: AxiosRequestConfig): Promise<T> => {
        const response = await axiosInstance.get<T>(url, config);
        return response.data;
    },

    post: async <T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> => {
        const response = await axiosInstance.post<T>(url, data, config);
        return response.data;
    },

    put: async <T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> => {
        const response = await axiosInstance.put<T>(url, data, config);
        return response.data;
    },

    patch: async <T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> => {
        const response = await axiosInstance.patch<T>(url, data, config);
        return response.data;
    },

    delete: async <T>(url: string, config?: AxiosRequestConfig): Promise<T> => {
        const response = await axiosInstance.delete<T>(url, config);
        return response.data;
    },
};

// Set auth token helper
export const setAuthToken = (token: string | null) => {
    if (token) {
        localStorage.setItem('api_key', token);
    } else {
        localStorage.removeItem('api_key');
    }
};

// Get auth token helper
export const getAuthToken = (): string | null => {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem('api_key');
};

export { axiosInstance };
export default api;
