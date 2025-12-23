import axios, { AxiosError, AxiosRequestConfig } from 'axios'
import type { ApiError } from '@/types'

// API base URL
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// Create axios instance
const axiosInstance = axios.create({
    baseURL: API_BASE_URL,
    headers: {
        'Content-Type': 'application/json',
    },
    timeout: 30000,
})

// Token getter - will be set by Clerk
let getToken: (() => Promise<string | null>) | null = null

export const setTokenGetter = (getter: () => Promise<string | null>) => {
    getToken = getter
}

// Request interceptor for adding auth token from Clerk
axiosInstance.interceptors.request.use(
    async (config) => {
        if (getToken) {
            try {
                const token = await getToken()
                if (token) {
                    config.headers.Authorization = `Bearer ${token}`
                }
            } catch (error) {
                console.warn('Failed to get auth token:', error)
            }
        }
        return config
    },
    (error) => Promise.reject(error)
)

// Response interceptor for error handling
axiosInstance.interceptors.response.use(
    (response) => response,
    (error: AxiosError<ApiError>) => {
        if (error.response) {
            const apiError: ApiError = {
                message: error.response.data?.message || 'An error occurred',
                code: error.response.data?.code,
                details: error.response.data?.details,
            }
            return Promise.reject(apiError)
        } else if (error.request) {
            return Promise.reject({
                message: 'Network error. Please check your connection.',
                code: 'NETWORK_ERROR',
            } as ApiError)
        } else {
            return Promise.reject({
                message: error.message,
                code: 'REQUEST_ERROR',
            } as ApiError)
        }
    }
)

// Generic API request functions
export const api = {
    get: async <T>(url: string, config?: AxiosRequestConfig): Promise<T> => {
        const response = await axiosInstance.get<T>(url, config)
        return response.data
    },

    post: async <T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> => {
        const response = await axiosInstance.post<T>(url, data, config)
        return response.data
    },

    put: async <T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> => {
        const response = await axiosInstance.put<T>(url, data, config)
        return response.data
    },

    patch: async <T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> => {
        const response = await axiosInstance.patch<T>(url, data, config)
        return response.data
    },

    delete: async <T>(url: string, config?: AxiosRequestConfig): Promise<T> => {
        const response = await axiosInstance.delete<T>(url, config)
        return response.data
    },
}

export { axiosInstance }
export default api
