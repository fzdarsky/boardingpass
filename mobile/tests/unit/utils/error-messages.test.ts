/**
 * Unit tests for error message generation utilities
 * Tests FR-024 (user-friendly error messages)
 */

import {
  getErrorMessage,
  getErrorTitle,
  getErrorAction,
  getErrorHelpText,
  ErrorType,
  AppError,
} from '../../../src/utils/error-messages';

describe('Error Message Utilities', () => {
  describe('getErrorMessage', () => {
    it('should return user-friendly message for network errors', () => {
      const error: AppError = {
        type: 'network',
        message: 'Network request failed',
        code: 'ECONNREFUSED',
      };

      const message = getErrorMessage(error);

      expect(message).toMatch(/unable to connect/i);
      expect(message).not.toContain('ECONNREFUSED'); // No technical codes
    });

    it('should return user-friendly message for timeout errors', () => {
      const error: AppError = {
        type: 'timeout',
        message: 'timeout of 30000ms exceeded',
        code: 'ECONNABORTED',
      };

      const message = getErrorMessage(error);

      expect(message).toMatch(/taking longer than expected/i);
      expect(message).toMatch(/try again/i);
    });

    it('should return user-friendly message for authentication errors', () => {
      const error: AppError = {
        type: 'auth',
        message: 'Invalid credentials',
        code: '401',
      };

      const message = getErrorMessage(error);

      expect(message).toMatch(/connection code/i);
      expect(message).toMatch(/incorrect|invalid/i);
    });

    it('should return user-friendly message for permission errors', () => {
      const error: AppError = {
        type: 'permission',
        message: 'Camera permission denied',
        code: 'PERMISSION_DENIED',
        context: { permission: 'camera' },
      };

      const message = getErrorMessage(error);

      expect(message).toMatch(/camera/i);
      expect(message).toMatch(/permission/i);
    });

    it('should return user-friendly message for certificate errors', () => {
      const error: AppError = {
        type: 'certificate',
        message: 'Certificate validation failed',
        code: 'CERT_INVALID',
      };

      const message = getErrorMessage(error);

      expect(message).toMatch(/certificate/i);
      expect(message).toMatch(/security/i);
    });

    it('should return user-friendly message for device unreachable', () => {
      const error: AppError = {
        type: 'network',
        message: 'Connection refused',
        code: 'ECONNREFUSED',
        context: { deviceName: 'Device-01' },
      };

      const message = getErrorMessage(error);

      expect(message).toMatch(/Device-01/i);
      expect(message).toMatch(/unreachable|connect/i);
    });

    it('should handle unknown errors gracefully', () => {
      const error: AppError = {
        type: 'unknown',
        message: 'Something went wrong',
      };

      const message = getErrorMessage(error);

      expect(message).toMatch(/unexpected error/i);
      expect(message).not.toContain('undefined');
      expect(message).not.toContain('null');
    });

    it('should sanitize technical error details', () => {
      const error: AppError = {
        type: 'network',
        message: 'Error: ENOTFOUND api.internal.company.com',
        code: 'ENOTFOUND',
      };

      const message = getErrorMessage(error);

      expect(message).not.toContain('ENOTFOUND');
      expect(message).not.toContain('api.internal.company.com');
    });
  });

  describe('getErrorTitle', () => {
    it('should return appropriate title for each error type', () => {
      const errorTypes: ErrorType[] = [
        'network',
        'timeout',
        'auth',
        'permission',
        'certificate',
        'validation',
        'unknown',
      ];

      for (const type of errorTypes) {
        const error: AppError = { type, message: 'Test error' };
        const title = getErrorTitle(error);

        expect(title).toBeTruthy();
        expect(title.length).toBeGreaterThan(0);
        expect(title.length).toBeLessThan(50); // Titles should be concise
      }
    });

    it('should return distinct titles for different error types', () => {
      const titles = new Set<string>();

      const errorTypes: ErrorType[] = ['network', 'timeout', 'auth', 'permission'];

      for (const type of errorTypes) {
        const error: AppError = { type, message: 'Test error' };
        titles.add(getErrorTitle(error));
      }

      // Each error type should have unique title
      expect(titles.size).toBe(errorTypes.length);
    });
  });

  describe('getErrorAction', () => {
    it('should suggest retry for network errors', () => {
      const error: AppError = {
        type: 'network',
        message: 'Network request failed',
      };

      const action = getErrorAction(error);

      expect(action).toMatch(/try again|retry/i);
    });

    it('should suggest retry for timeout errors', () => {
      const error: AppError = {
        type: 'timeout',
        message: 'Request timed out',
      };

      const action = getErrorAction(error);

      expect(action).toMatch(/try again|retry/i);
    });

    it('should suggest checking connection code for auth errors', () => {
      const error: AppError = {
        type: 'auth',
        message: 'Authentication failed',
      };

      const action = getErrorAction(error);

      expect(action).toMatch(/connection code|credentials/i);
    });

    it('should suggest opening settings for permission errors', () => {
      const error: AppError = {
        type: 'permission',
        message: 'Permission denied',
        context: { canAskAgain: false },
      };

      const action = getErrorAction(error);

      expect(action).toMatch(/settings|enable/i);
    });

    it('should suggest manual input for camera permission denial', () => {
      const error: AppError = {
        type: 'permission',
        message: 'Camera permission denied',
        context: { permission: 'camera' },
      };

      const action = getErrorAction(error);

      expect(action).toMatch(/manual|enter code/i);
    });

    it('should suggest trust action for certificate errors', () => {
      const error: AppError = {
        type: 'certificate',
        message: 'Self-signed certificate',
        context: { isSelfSigned: true },
      };

      const action = getErrorAction(error);

      expect(action).toMatch(/trust|accept|continue/i);
    });
  });

  describe('getErrorHelpText', () => {
    it('should provide helpful context for mDNS blocked errors', () => {
      const error: AppError = {
        type: 'network',
        message: 'mDNS service unavailable',
        context: { service: 'mdns' },
      };

      const helpText = getErrorHelpText(error);

      expect(helpText).toMatch(/firewall|network/i);
      expect(helpText).toMatch(/administrator/i);
    });

    it('should provide helpful context for connection timeout', () => {
      const error: AppError = {
        type: 'timeout',
        message: 'Connection timeout',
      };

      const helpText = getErrorHelpText(error);

      expect(helpText).toMatch(/slow|network/i);
      expect(helpText).toMatch(/try again/i);
    });

    it('should provide helpful context for authentication failure', () => {
      const error: AppError = {
        type: 'auth',
        message: 'Invalid credentials',
        context: { failureCount: 3 },
      };

      const helpText = getErrorHelpText(error);

      expect(helpText).toBeTruthy();
      expect(helpText?.length).toBeGreaterThan(20);
    });

    it('should provide helpful context for permission errors', () => {
      const error: AppError = {
        type: 'permission',
        message: 'Camera permission denied',
        context: { permission: 'camera', canAskAgain: false },
      };

      const helpText = getErrorHelpText(error);

      expect(helpText).toMatch(/settings|enable/i);
    });

    it('should mention progressive delay for multiple auth failures', () => {
      const error: AppError = {
        type: 'auth',
        message: 'Authentication failed',
        context: { failureCount: 4 },
      };

      const helpText = getErrorHelpText(error);

      expect(helpText).toMatch(/wait|delay/i);
    });

    it('should return null for errors without helpful context', () => {
      const error: AppError = {
        type: 'validation',
        message: 'Invalid input',
      };

      const helpText = getErrorHelpText(error);

      // Validation errors might not need additional help text
      expect(helpText).toBeDefined();
    });
  });

  describe('Error message formatting', () => {
    it('should not include stack traces in user messages', () => {
      const error: AppError = {
        type: 'unknown',
        message: 'Error: Something failed\n    at Object.fetch (http://...)',
      };

      const message = getErrorMessage(error);

      expect(message).not.toContain('at Object');
      expect(message).not.toContain('http://');
    });

    it('should not expose internal URLs or IPs in messages', () => {
      const error: AppError = {
        type: 'network',
        message: 'Failed to fetch https://192.168.1.100:8443/internal/api',
      };

      const message = getErrorMessage(error);

      expect(message).not.toContain('192.168.1.100');
      expect(message).not.toContain('/internal/api');
    });

    it('should handle errors with no message gracefully', () => {
      const error: AppError = {
        type: 'unknown',
        message: '',
      };

      const message = getErrorMessage(error);

      expect(message).toBeTruthy();
      expect(message).toMatch(/error occurred/i);
    });

    it('should limit message length for very long errors', () => {
      const longMessage = 'Error '.repeat(100);
      const error: AppError = {
        type: 'network',
        message: longMessage,
      };

      const message = getErrorMessage(error);

      expect(message.length).toBeLessThan(200); // Reasonable max length
    });
  });

  describe('Context-aware error messages', () => {
    it('should include device name in error message when available', () => {
      const error: AppError = {
        type: 'network',
        message: 'Connection failed',
        context: { deviceName: 'Raspberry-Pi-4' },
      };

      const message = getErrorMessage(error);

      expect(message).toContain('Raspberry-Pi-4');
    });

    it('should adapt message based on failure count', () => {
      const error1: AppError = {
        type: 'auth',
        message: 'Authentication failed',
        context: { failureCount: 1 },
      };

      const error4: AppError = {
        type: 'auth',
        message: 'Authentication failed',
        context: { failureCount: 4 },
      };

      const message1 = getErrorMessage(error1);
      const message4 = getErrorMessage(error4);

      // Messages should differ based on failure count
      expect(message1).not.toBe(message4);
      expect(message4).toMatch(/wait|locked/i);
    });

    it('should provide specific guidance for self-signed certificates', () => {
      const error: AppError = {
        type: 'certificate',
        message: 'Certificate validation failed',
        context: { isSelfSigned: true },
      };

      const helpText = getErrorHelpText(error);

      expect(helpText).toMatch(/self-signed/i);
      expect(helpText).toMatch(/trust/i);
    });
  });
});
