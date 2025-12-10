/**
 * Error Utilities
 *
 * Custom error types and error handling helpers for the BoardingPass mobile app.
 */

/**
 * Base application error
 */
export class AppError extends Error {
  constructor(
    message: string,
    public code?: string,
    public userMessage?: string
  ) {
    super(message);
    this.name = 'AppError';
  }
}

/**
 * Network-related errors
 */
export class NetworkError extends AppError {
  constructor(message: string, code?: string) {
    super(message, code, 'Network connection failed');
    this.name = 'NetworkError';
  }
}

/**
 * Authentication errors
 */
export class AuthenticationError extends AppError {
  constructor(message: string, code?: string) {
    super(message, code, 'Authentication failed');
    this.name = 'AuthenticationError';
  }
}

/**
 * Certificate validation errors
 */
export class CertificateError extends AppError {
  constructor(message: string, code?: string) {
    super(message, code, 'Certificate validation failed');
    this.name = 'CertificateError';
  }
}

/**
 * Device discovery errors
 */
export class DiscoveryError extends AppError {
  constructor(message: string, code?: string) {
    super(message, code, 'Device discovery failed');
    this.name = 'DiscoveryError';
  }
}

/**
 * Validation errors
 */
export class ValidationError extends AppError {
  constructor(
    message: string,
    public field?: string
  ) {
    super(message, 'VALIDATION_ERROR', 'Invalid input');
    this.name = 'ValidationError';
  }
}

/**
 * Timeout errors
 */
export class TimeoutError extends AppError {
  constructor(message: string = 'Request timed out') {
    super(message, 'TIMEOUT', 'Request timed out');
    this.name = 'TimeoutError';
  }
}

/**
 * Permission errors
 */
export class PermissionError extends AppError {
  constructor(
    message: string,
    public permission: string
  ) {
    super(message, 'PERMISSION_DENIED', 'Permission denied');
    this.name = 'PermissionError';
  }
}

/**
 * Error type guards
 */
export function isNetworkError(error: unknown): error is NetworkError {
  return error instanceof NetworkError;
}

export function isAuthenticationError(error: unknown): error is AuthenticationError {
  return error instanceof AuthenticationError;
}

export function isCertificateError(error: unknown): error is CertificateError {
  return error instanceof CertificateError;
}

export function isValidationError(error: unknown): error is ValidationError {
  return error instanceof ValidationError;
}

export function isTimeoutError(error: unknown): error is TimeoutError {
  return error instanceof TimeoutError;
}

export function isPermissionError(error: unknown): error is PermissionError {
  return error instanceof PermissionError;
}

/**
 * Get user-friendly error message
 */
export function getUserErrorMessage(error: unknown): string {
  if (error instanceof AppError && error.userMessage) {
    return error.userMessage;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return 'An unexpected error occurred';
}

/**
 * Format error for logging (excludes sensitive data)
 */
export function formatErrorForLogging(error: unknown): Record<string, string | undefined> {
  if (error instanceof AppError) {
    return {
      name: error.name,
      code: error.code,
      message: error.message,
      // Explicitly exclude any sensitive fields
    };
  }

  if (error instanceof Error) {
    return {
      name: error.name,
      message: error.message,
      stack: error.stack,
    };
  }

  return {
    error: String(error),
  };
}

/**
 * Retry configuration
 */
export interface RetryConfig {
  maxAttempts: number;
  delayMs: number;
  backoffMultiplier?: number;
  shouldRetry?: (error: unknown) => boolean;
}

/**
 * Retry a function with exponential backoff
 */
export async function retryWithBackoff<T>(fn: () => Promise<T>, config: RetryConfig): Promise<T> {
  let lastError: unknown;
  let delay = config.delayMs;

  for (let attempt = 1; attempt <= config.maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;

      // Check if we should retry this error
      if (config.shouldRetry && !config.shouldRetry(error)) {
        throw error;
      }

      // Don't retry on last attempt
      if (attempt === config.maxAttempts) {
        break;
      }

      // Wait before retrying
      await new Promise(resolve => setTimeout(resolve, delay));

      // Exponential backoff
      if (config.backoffMultiplier) {
        delay *= config.backoffMultiplier;
      }
    }
  }

  throw lastError;
}

/**
 * Default retry configuration for network requests
 */
export const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxAttempts: 3,
  delayMs: 1000,
  backoffMultiplier: 2,
  shouldRetry: (error: unknown) => {
    // Retry on network errors and timeouts, but not on auth/validation errors
    return (
      isNetworkError(error) ||
      isTimeoutError(error) ||
      (error instanceof Error && error.message.includes('Network'))
    );
  },
};

/**
 * Handle error with user-friendly message and logging
 */
export function handleError(error: unknown, context?: string): void {
  // Log error (without sensitive data)
  const errorLog = formatErrorForLogging(error);
  console.error(`Error${context ? ` in ${context}` : ''}:`, errorLog);

  // Could integrate with error tracking service here
  // e.g., Sentry.captureException(error);
}
