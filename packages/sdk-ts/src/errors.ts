/**
 * Structured error handling for SimpleEmailAPI SDK.
 * Provides typed error codes for programmatic error handling.
 * 
 * @example
 * ```ts
 * import { ErrorCode, parseError } from 'simpleemailapi'
 * 
 * try {
 *   await client.send({ ... })
 * } catch (err) {
 *   const error = parseError(err)
 *   if (error.is(ErrorCode.DOMAIN_NOT_VERIFIED)) {
 *     console.log('Please verify your domain first')
 *   }
 * }
 * ```
 */

import { ConnectError, Code } from '@connectrpc/connect'
import { ErrorCode, ErrorDetailSchema } from './gen/v1/errors_pb'
import type { ErrorDetail } from './gen/v1/errors_pb'

// Re-export for easy access
export { ErrorCode } from './gen/v1/errors_pb'

/**
 * Typed error class for SimpleEmailAPI errors.
 * Provides typed access to error codes for programmatic handling.
 */
export class SimpleEmailError extends Error {
    /** The specific error code from the API. */
    readonly code: ErrorCode

    /** The field that caused the error (for validation errors). */
    readonly field?: string

    /** Additional context (e.g., limits, upgrade URLs). */
    readonly metadata: Record<string, string>

    /** The underlying gRPC status code. */
    readonly grpcCode: Code

    constructor(
        connectError: ConnectError,
        detail?: ErrorDetail
    ) {
        super(detail?.message ?? connectError.message)
        this.name = 'SimpleEmailError'
        this.grpcCode = connectError.code
        this.code = detail?.code ?? ErrorCode.UNSPECIFIED
        this.field = detail?.field || undefined
        this.metadata = detail?.metadata ?? {}
    }

    /**
     * Check if this error matches a specific error code.
     * @example
     * if (error.is(ErrorCode.DOMAIN_NOT_VERIFIED)) {
     *   // Prompt user to verify domain
     * }
     */
    is(code: ErrorCode): boolean {
        return this.code === code
    }

    /**
     * Check if this error is in a specific category.
     * @example
     * if (error.isCategory('validation')) {
     *   // Handle validation errors
     * }
     */
    isCategory(category: 'auth' | 'authz' | 'validation' | 'notfound' | 'domain' | 'ratelimit' | 'internal'): boolean {
        const c = this.code
        switch (category) {
            case 'auth': return c >= 100 && c < 200
            case 'authz': return c >= 200 && c < 300
            case 'validation': return c >= 300 && c < 400
            case 'notfound': return c >= 400 && c < 410
            case 'domain': return c >= 500 && c < 600
            case 'ratelimit': return c >= 600 && c < 700
            case 'internal': return c >= 900
            default: return false
        }
    }
}

/**
 * Parse any error into a typed SimpleEmailError.
 * Extracts ErrorDetail from Connect RPC errors if present.
 */
export function parseError(err: unknown): SimpleEmailError {
    if (err instanceof SimpleEmailError) {
        return err
    }

    if (err instanceof ConnectError) {
        // Try to extract ErrorDetail from error details using the schema
        try {
            const details = err.findDetails(ErrorDetailSchema)
            const detail = details.length > 0 ? details[0] : undefined
            return new SimpleEmailError(err, detail)
        } catch {
            return new SimpleEmailError(err)
        }
    }

    // Wrap unknown errors
    const message = err instanceof Error ? err.message : String(err)
    const connectErr = new ConnectError(message, Code.Unknown)
    return new SimpleEmailError(connectErr)
}

/**
 * Type guard for SimpleEmailError.
 */
export function isSimpleEmailError(err: unknown): err is SimpleEmailError {
    return err instanceof SimpleEmailError
}
