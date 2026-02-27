/**
 * Error message utilities for user-friendly error display
 * Implements FR-024 (clear error messages)
 */

export type ErrorType =
  | 'network'
  | 'timeout'
  | 'auth'
  | 'permission'
  | 'certificate'
  | 'validation'
  | 'unknown';

export interface AppError {
  type: ErrorType;
  message: string;
  code?: string;
  context?: Record<string, any>;
}

/**
 * Convert technical error message to user-friendly message
 */
export function getErrorMessage(error: AppError): string {
  const { type } = error;

  switch (type) {
    case 'network':
      return getNetworkErrorMessage(error);

    case 'timeout':
      return getTimeoutErrorMessage(error);

    case 'auth':
      return getAuthErrorMessage(error);

    case 'permission':
      return getPermissionErrorMessage(error);

    case 'certificate':
      return getCertificateErrorMessage(error);

    case 'validation':
      return getValidationErrorMessage(error);

    case 'unknown':
    default:
      return 'An unexpected error occurred. Please try again.';
  }
}

/**
 * Get user-friendly error title for display in UI
 */
export function getErrorTitle(error: AppError): string {
  const { type } = error;

  switch (type) {
    case 'network':
      return 'Connection Problem';

    case 'timeout':
      return 'Request Timed Out';

    case 'auth':
      return 'Authentication Failed';

    case 'permission':
      return 'Permission Required';

    case 'certificate':
      return 'Security Certificate';

    case 'validation':
      return 'Invalid Input';

    case 'unknown':
    default:
      return 'Error';
  }
}

/**
 * Get recommended action for error recovery
 */
export function getErrorAction(error: AppError): string {
  const { type, context } = error;

  switch (type) {
    case 'network':
    case 'timeout':
      return 'Try Again';

    case 'auth':
      if (context?.failureCount >= 4) {
        return 'Wait and Try Again';
      }
      return 'Check Connection Code';

    case 'permission':
      if (context?.canAskAgain === false) {
        return 'Open Settings';
      }
      if (context?.permission === 'camera') {
        return 'Enter Code Manually';
      }
      return 'Grant Permission';

    case 'certificate':
      if (context?.isSelfSigned) {
        return 'Trust Certificate';
      }
      return 'Cancel';

    case 'validation':
      return 'Correct and Retry';

    case 'unknown':
    default:
      return 'Dismiss';
  }
}

/**
 * Get additional help text for error context
 */
export function getErrorHelpText(error: AppError): string | null {
  const { type, context } = error;

  switch (type) {
    case 'network':
      if (context?.service === 'mdns') {
        return 'Device discovery requires local network access. Check firewall settings or contact your network administrator.';
      }
      if (context?.deviceName) {
        return `Make sure ${context.deviceName} is powered on and connected to the same network.`;
      }
      return 'Check your network connection and try again.';

    case 'timeout':
      return 'The request is taking longer than expected. This could be due to slow network or device being busy. Try again or check your network connection.';

    case 'auth':
      if (context?.failureCount >= 4) {
        return 'Too many failed attempts. Please wait 60 seconds before trying again.';
      }
      if (context?.failureCount >= 2) {
        return 'Multiple failed attempts detected. Double-check your connection code.';
      }
      return 'The connection code may be incorrect or expired. Request a new code from the device.';

    case 'permission':
      if (context?.permission === 'camera') {
        if (context?.canAskAgain === false) {
          return 'Camera access is required for QR scanning. Open Settings to enable camera permission, or enter the connection code manually.';
        }
        return 'Camera access is required to scan QR codes. You can also enter the connection code manually.';
      }
      if (context?.permission === 'local_network') {
        return 'Local network access is required to discover devices on your network.';
      }
      return 'This feature requires additional permissions to function.';

    case 'certificate':
      if (context?.isSelfSigned) {
        return 'This device uses a self-signed certificate. Trusting it will allow encrypted communication but cannot verify the certificate authority.';
      }
      if (context?.changed) {
        return 'The device certificate has changed since your last connection. This could indicate a security risk.';
      }
      return 'Certificate validation failed. Connection may not be secure.';

    case 'validation':
      return 'Please check your input and try again.';

    case 'unknown':
    default:
      return null;
  }
}

// Type-specific message generators

function getNetworkErrorMessage(error: AppError): string {
  const { code, context } = error;

  if (code === 'ECONNREFUSED' || code === 'ECONNRESET') {
    const deviceName = context?.deviceName || 'the device';
    return `Unable to connect to ${deviceName}. Make sure it's powered on and connected to the same network.`;
  }

  if (code === 'ENOTFOUND') {
    return 'Device not found on the network. It may have disconnected or changed IP address.';
  }

  if (code === 'NETWORK_ERROR' || error.message.includes('Network request failed')) {
    return 'Network connection failed. Check your WiFi or mobile data connection.';
  }

  if (context?.service === 'mdns') {
    return 'Unable to discover devices. Check if local network access is allowed.';
  }

  // Include device name in generic message if available
  if (context?.deviceName) {
    return `Unable to connect to ${context.deviceName}. Check your network connection and try again.`;
  }

  return 'Unable to connect. Check your network connection and try again.';
}

function getTimeoutErrorMessage(error: AppError): string {
  const { context } = error;

  if (context?.operation === 'discovery') {
    return 'Device discovery is taking longer than expected. Try again or enter device details manually.';
  }

  if (context?.operation === 'auth') {
    return 'Authentication is taking longer than expected. The device may be busy.';
  }

  if (context?.operation === 'info') {
    return 'Loading device information is taking longer than expected.';
  }

  return 'The request is taking longer than expected. Try again or check your network connection.';
}

function getAuthErrorMessage(error: AppError): string {
  const { code, context } = error;

  if (code === '401' || code === 'INVALID_CREDENTIALS') {
    if (context?.failureCount >= 4) {
      return 'Too many failed authentication attempts. Please wait 60 seconds before trying again.';
    }
    return 'The connection code is incorrect or has expired. Please check and try again.';
  }

  if (code === '403') {
    return 'Access denied. This device may not be accepting new connections.';
  }

  if (code === 'TOKEN_EXPIRED') {
    return 'Your session has expired. Please authenticate again.';
  }

  // Different messages based on failure count
  if (context?.failureCount >= 4) {
    return 'Too many failed attempts. You must wait before trying again.';
  }

  return 'Authentication failed. Check your connection code and try again.';
}

function getPermissionErrorMessage(error: AppError): string {
  const { context } = error;

  if (context?.permission === 'camera') {
    if (context?.canAskAgain === false) {
      return 'Camera permission is required for QR scanning. Please enable it in Settings, or enter the connection code manually.';
    }
    return 'Camera permission is required to scan QR codes. Grant permission or enter the code manually.';
  }

  if (context?.permission === 'local_network') {
    return 'Local network access is required to discover BoardingPass devices on your network.';
  }

  return 'This feature requires additional permissions. Please grant the necessary permissions to continue.';
}

function getCertificateErrorMessage(error: AppError): string {
  const { context } = error;

  if (context?.isSelfSigned && !context?.changed) {
    return 'This device uses a self-signed security certificate. Review the certificate details and confirm if you trust this device.';
  }

  if (context?.changed) {
    return 'Security Alert: The device certificate has changed since your last connection. Only proceed if you expect this change.';
  }

  if (context?.expired) {
    return 'The device security certificate has expired. Connection may not be secure.';
  }

  return 'Certificate validation failed. This is a security issue and the connection may not be secure.';
}

function getValidationErrorMessage(error: AppError): string {
  const { context } = error;

  if (context?.field === 'connectionCode') {
    return 'The connection code format is invalid. It should be a base64-encoded string.';
  }

  if (context?.field === 'ipAddress') {
    return 'Invalid IP address. Please enter a valid IPv4 or IPv6 address.';
  }

  if (context?.field === 'port') {
    return 'Invalid port number. Port must be between 1 and 65535.';
  }

  return 'Invalid input. Please check your entry and try again.';
}

/**
 * Create AppError from unknown error object
 */
export function toAppError(error: unknown, type: ErrorType = 'unknown'): AppError {
  if (typeof error === 'object' && error !== null) {
    const err = error as any;

    return {
      type: err.type || type,
      message: err.message || 'An error occurred',
      code: err.code || err.status?.toString(),
      context: err.context || {},
    };
  }

  return {
    type,
    message: String(error) || 'An error occurred',
  };
}

/**
 * Check if error is a network error
 */
export function isNetworkError(error: unknown): boolean {
  if (typeof error === 'object' && error !== null) {
    const err = error as any;
    const code = err.code || '';
    const message = err.message || '';

    return (
      code.startsWith('E') ||
      message.includes('Network') ||
      message.includes('network') ||
      code === 'NETWORK_ERROR'
    );
  }

  return false;
}

/**
 * Check if error is a timeout error
 */
export function isTimeoutError(error: unknown): boolean {
  if (typeof error === 'object' && error !== null) {
    const err = error as any;
    const code = err.code || '';
    const message = err.message || '';

    return (
      code === 'ECONNABORTED' ||
      code === 'ETIMEDOUT' ||
      message.includes('timeout') ||
      message.includes('timed out')
    );
  }

  return false;
}

/**
 * Check if error is an authentication error
 */
export function isAuthError(error: unknown): boolean {
  if (typeof error === 'object' && error !== null) {
    const err = error as any;
    const status = err.status || err.response?.status;

    return status === 401 || status === 403 || err.type === 'auth';
  }

  return false;
}
