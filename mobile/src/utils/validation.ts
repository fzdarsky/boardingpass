/**
 * Validation Utilities
 *
 * Input validation functions for IP addresses, connection codes, fingerprints, etc.
 */

/**
 * IPv4 address validation
 */
export function isValidIPv4(ip: string): boolean {
  const ipv4Pattern =
    /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
  return ipv4Pattern.test(ip);
}

/**
 * IPv6 address validation (simplified - covers most cases)
 */
export function isValidIPv6(ip: string): boolean {
  const ipv6Pattern =
    /^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$/;
  return ipv6Pattern.test(ip);
}

/**
 * IP address validation (IPv4 or IPv6)
 */
export function isValidIPAddress(ip: string): boolean {
  return isValidIPv4(ip) || isValidIPv6(ip);
}

/**
 * Hostname validation (RFC 1123)
 */
export function isValidHostname(hostname: string): boolean {
  // RFC 1123: alphanumeric + hyphens, max 253 chars total, labels max 63 chars
  const hostnamePattern = /^(?!-)[A-Za-z0-9-]{1,63}(?<!-)(\.[A-Za-z0-9-]{1,63})*$/;
  return hostname.length <= 253 && hostnamePattern.test(hostname);
}

/**
 * Host validation (IP address or hostname)
 */
export function isValidHost(host: string): boolean {
  return isValidIPAddress(host) || isValidHostname(host);
}

/**
 * Port number validation
 */
export function isValidPort(port: number): boolean {
  return Number.isInteger(port) && port >= 1 && port <= 65535;
}

/**
 * Connection code validation
 *
 * Format: Base64-encoded string, minimum 32 characters
 * Exact format may vary - this is a permissive check
 */
export function isValidConnectionCode(code: string): boolean {
  // Allow base64 characters: alphanumeric, +, /, =
  const base64Pattern = /^[A-Za-z0-9+/=]{32,}$/;
  return base64Pattern.test(code.trim());
}

/**
 * Certificate fingerprint validation
 *
 * SHA-256 fingerprint: 64 hexadecimal characters
 */
export function isValidFingerprint(fingerprint: string): boolean {
  const fingerprintPattern = /^[a-fA-F0-9]{64}$/;
  return fingerprintPattern.test(fingerprint);
}

/**
 * MAC address validation
 *
 * Format: XX:XX:XX:XX:XX:XX (colon-separated hex pairs)
 */
export function isValidMACAddress(mac: string): boolean {
  const macPattern = /^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$/;
  return macPattern.test(mac);
}

/**
 * UUID validation (RFC 4122)
 */
export function isValidUUID(uuid: string): boolean {
  const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
  return uuidPattern.test(uuid);
}

/**
 * Device name validation
 */
export function isValidDeviceName(name: string): boolean {
  return name.trim().length > 0 && name.length <= 255;
}

/**
 * URL validation (simplified - for mDNS service URLs)
 */
export function isValidURL(url: string): boolean {
  try {
    const _ = new URL(url);
    return true;
  } catch {
    return false;
  }
}

/**
 * Sanitize user input (remove dangerous characters)
 */
export function sanitizeInput(input: string): string {
  // Remove control characters and trim
  // eslint-disable-next-line no-control-regex
  return input.replace(/[\x00-\x1F\x7F]/g, '').trim();
}

/**
 * Validate and sanitize connection code input
 */
export function validateAndSanitizeConnectionCode(code: string): {
  valid: boolean;
  sanitized: string;
  error?: string;
} {
  const sanitized = sanitizeInput(code);

  if (sanitized.length === 0) {
    return { valid: false, sanitized: '', error: 'Connection code cannot be empty' };
  }

  if (!isValidConnectionCode(sanitized)) {
    return {
      valid: false,
      sanitized,
      error: 'Invalid connection code format',
    };
  }

  return { valid: true, sanitized };
}
