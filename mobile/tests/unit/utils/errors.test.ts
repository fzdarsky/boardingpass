/**
 * Error Utilities Tests
 */

import {
  AppError,
  NetworkError,
  AuthenticationError,
  CertificateError,
  ValidationError,
  TimeoutError,
  PermissionError,
  isNetworkError,
  isAuthenticationError,
  isCertificateError,
  isValidationError,
  isTimeoutError,
  isPermissionError,
  getUserErrorMessage,
  formatErrorForLogging,
  retryWithBackoff,
  DEFAULT_RETRY_CONFIG,
} from '../../../src/utils/errors';

describe('Error Classes', () => {
  describe('AppError', () => {
    it('should create error with message and code', () => {
      const error = new AppError('Test error', 'TEST_CODE', 'User message');
      expect(error.message).toBe('Test error');
      expect(error.code).toBe('TEST_CODE');
      expect(error.userMessage).toBe('User message');
      expect(error.name).toBe('AppError');
    });
  });

  describe('NetworkError', () => {
    it('should create network error', () => {
      const error = new NetworkError('Network failed', 'NET_ERR');
      expect(error.message).toBe('Network failed');
      expect(error.code).toBe('NET_ERR');
      expect(error.userMessage).toBe('Network connection failed');
      expect(error.name).toBe('NetworkError');
    });
  });

  describe('AuthenticationError', () => {
    it('should create authentication error', () => {
      const error = new AuthenticationError('Auth failed', 'AUTH_ERR');
      expect(error.message).toBe('Auth failed');
      expect(error.userMessage).toBe('Authentication failed');
      expect(error.name).toBe('AuthenticationError');
    });
  });

  describe('CertificateError', () => {
    it('should create certificate error', () => {
      const error = new CertificateError('Cert invalid', 'CERT_ERR');
      expect(error.message).toBe('Cert invalid');
      expect(error.userMessage).toBe('Certificate validation failed');
      expect(error.name).toBe('CertificateError');
    });
  });

  describe('ValidationError', () => {
    it('should create validation error with field', () => {
      const error = new ValidationError('Invalid input', 'email');
      expect(error.message).toBe('Invalid input');
      expect(error.field).toBe('email');
      expect(error.code).toBe('VALIDATION_ERROR');
      expect(error.name).toBe('ValidationError');
    });
  });

  describe('TimeoutError', () => {
    it('should create timeout error', () => {
      const error = new TimeoutError();
      expect(error.message).toBe('Request timed out');
      expect(error.code).toBe('TIMEOUT');
      expect(error.name).toBe('TimeoutError');
    });
  });

  describe('PermissionError', () => {
    it('should create permission error', () => {
      const error = new PermissionError('Camera denied', 'camera');
      expect(error.message).toBe('Camera denied');
      expect(error.permission).toBe('camera');
      expect(error.code).toBe('PERMISSION_DENIED');
      expect(error.name).toBe('PermissionError');
    });
  });
});

describe('Error Type Guards', () => {
  it('should identify network errors', () => {
    const error = new NetworkError('test');
    expect(isNetworkError(error)).toBe(true);
    expect(isAuthenticationError(error)).toBe(false);
  });

  it('should identify authentication errors', () => {
    const error = new AuthenticationError('test');
    expect(isAuthenticationError(error)).toBe(true);
    expect(isNetworkError(error)).toBe(false);
  });

  it('should identify certificate errors', () => {
    const error = new CertificateError('test');
    expect(isCertificateError(error)).toBe(true);
    expect(isValidationError(error)).toBe(false);
  });

  it('should identify validation errors', () => {
    const error = new ValidationError('test');
    expect(isValidationError(error)).toBe(true);
    expect(isTimeoutError(error)).toBe(false);
  });

  it('should identify timeout errors', () => {
    const error = new TimeoutError();
    expect(isTimeoutError(error)).toBe(true);
    expect(isPermissionError(error)).toBe(false);
  });

  it('should identify permission errors', () => {
    const error = new PermissionError('test', 'camera');
    expect(isPermissionError(error)).toBe(true);
    expect(isNetworkError(error)).toBe(false);
  });
});

describe('Error Message Helpers', () => {
  describe('getUserErrorMessage', () => {
    it('should return user message from AppError', () => {
      const error = new AppError('Internal error', 'CODE', 'User-friendly message');
      expect(getUserErrorMessage(error)).toBe('User-friendly message');
    });

    it('should return error message for regular Error', () => {
      const error = new Error('Standard error');
      expect(getUserErrorMessage(error)).toBe('Standard error');
    });

    it('should return generic message for unknown errors', () => {
      expect(getUserErrorMessage('string error')).toBe('An unexpected error occurred');
      expect(getUserErrorMessage(null)).toBe('An unexpected error occurred');
    });
  });

  describe('formatErrorForLogging', () => {
    it('should format AppError for logging', () => {
      const error = new AppError('Test error', 'TEST_CODE');
      const formatted = formatErrorForLogging(error);
      expect(formatted.name).toBe('AppError');
      expect(formatted.code).toBe('TEST_CODE');
      expect(formatted.message).toBe('Test error');
    });

    it('should format regular Error for logging', () => {
      const error = new Error('Test error');
      const formatted = formatErrorForLogging(error);
      expect(formatted.name).toBe('Error');
      expect(formatted.message).toBe('Test error');
      expect(formatted.stack).toBeDefined();
    });

    it('should format unknown errors', () => {
      const formatted = formatErrorForLogging('string error');
      expect(formatted.error).toBe('string error');
    });
  });
});

describe('Retry with Backoff', () => {
  it('should retry failed operations', async () => {
    let attempts = 0;
    const fn = jest.fn(async () => {
      attempts++;
      if (attempts < 3) {
        throw new NetworkError('Network failed');
      }
      return 'success';
    });

    const result = await retryWithBackoff(fn, {
      maxAttempts: 3,
      delayMs: 10,
      backoffMultiplier: 2,
    });

    expect(result).toBe('success');
    expect(attempts).toBe(3);
    expect(fn).toHaveBeenCalledTimes(3);
  });

  it('should throw after max attempts', async () => {
    const fn = jest.fn(async () => {
      throw new NetworkError('Always fails');
    });

    await expect(
      retryWithBackoff(fn, {
        maxAttempts: 3,
        delayMs: 10,
      })
    ).rejects.toThrow('Always fails');

    expect(fn).toHaveBeenCalledTimes(3);
  });

  it('should not retry if shouldRetry returns false', async () => {
    const fn = jest.fn(async () => {
      throw new AuthenticationError('Auth failed');
    });

    await expect(
      retryWithBackoff(fn, {
        maxAttempts: 3,
        delayMs: 10,
        shouldRetry: error => !isAuthenticationError(error),
      })
    ).rejects.toThrow('Auth failed');

    expect(fn).toHaveBeenCalledTimes(1); // No retries
  });

  it('should apply exponential backoff', async () => {
    let attempts = 0;

    const fn = jest.fn(async () => {
      attempts++;
      if (attempts < 3) {
        throw new NetworkError('Network failed');
      }
      return 'success';
    });

    const startTime = Date.now();
    await retryWithBackoff(fn, {
      maxAttempts: 3,
      delayMs: 100,
      backoffMultiplier: 2,
    });
    const totalTime = Date.now() - startTime;

    // Total delay should be approximately: 100ms + 200ms = 300ms
    expect(totalTime).toBeGreaterThanOrEqual(250); // Allow some margin
  });
});

describe('DEFAULT_RETRY_CONFIG', () => {
  it('should retry on network errors', () => {
    const networkError = new NetworkError('test');
    expect(DEFAULT_RETRY_CONFIG.shouldRetry?.(networkError)).toBe(true);
  });

  it('should retry on timeout errors', () => {
    const timeoutError = new TimeoutError();
    expect(DEFAULT_RETRY_CONFIG.shouldRetry?.(timeoutError)).toBe(true);
  });

  it('should not retry on authentication errors', () => {
    const authError = new AuthenticationError('test');
    expect(DEFAULT_RETRY_CONFIG.shouldRetry?.(authError)).toBe(false);
  });

  it('should not retry on validation errors', () => {
    const validationError = new ValidationError('test');
    expect(DEFAULT_RETRY_CONFIG.shouldRetry?.(validationError)).toBe(false);
  });
});
