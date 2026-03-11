/**
 * Validation Utilities
 *
 * Input validation functions for IP addresses, connection codes, fingerprints, etc.
 */

import { DEFAULT_BOARDINGPASS_PORT } from '@/constants/network';

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
 * Accepts common formats:
 * - MAC address: XX:XX:XX:XX:XX:XX or XX-XX-XX-XX-XX-XX (typical device-generated code)
 * - Bare hex MAC: XXXXXXXXXXXX (12 hex chars, as scanned from barcodes)
 * - Base64 string: 32+ chars (alternative encoding)
 *
 * This is a permissive client-side gate; the server validates
 * the actual credential during SRP-6a authentication.
 */
export function isValidConnectionCode(code: string): boolean {
  const trimmed = code.trim();
  if (trimmed.length === 0) return false;

  // MAC address format (colon-separated hex pairs)
  if (/^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$/.test(trimmed)) return true;

  // MAC address format (dash-separated hex pairs)
  if (/^([0-9A-Fa-f]{2}-){5}[0-9A-Fa-f]{2}$/.test(trimmed)) return true;

  // Bare hex MAC (12 hex characters, no separators — common from barcode scans)
  if (/^[0-9A-Fa-f]{12}$/.test(trimmed)) return true;

  // Base64-encoded string (32+ chars)
  if (/^[A-Za-z0-9+/=]{32,}$/.test(trimmed)) return true;

  return false;
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
    // eslint-disable-next-line no-new
    new URL(url);
    return true;
  } catch {
    return false;
  }
}

/**
 * IPv4 address with optional port validation pattern
 * Matches: 192.168.1.100 or 192.168.1.100:9455
 */
const IPV4_WITH_PORT_PATTERN = /^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}(:\d{1,5})?$/;

/**
 * Parse and validate a host:port string for manual device entry.
 *
 * Accepted formats:
 *   "192.168.1.100"       → { host: "192.168.1.100", port: 9455 }
 *   "192.168.1.100:9443"  → { host: "192.168.1.100", port: 9443 }
 */
export function parseAndValidateHostPort(
  input: string,
  defaultPort: number = DEFAULT_BOARDINGPASS_PORT
): { valid: boolean; host?: string; port?: number; error?: string } {
  const trimmed = sanitizeInput(input);

  if (trimmed.length === 0) {
    return { valid: false, error: 'Address cannot be empty' };
  }

  // Validate against IPv4 pattern with optional port
  if (!IPV4_WITH_PORT_PATTERN.test(trimmed)) {
    return { valid: false, error: 'Invalid IPv4 address format' };
  }

  // Extract host and port
  const colonIndex = trimmed.lastIndexOf(':');
  let host: string;
  let port: number = defaultPort;

  // Check if there's a port (colon after the IP, not part of it)
  // IPv4 has exactly 3 dots, so colon after 4th octet is the port separator
  const dotCount = (trimmed.match(/\./g) || []).length;
  if (colonIndex > -1 && dotCount === 3) {
    const afterLastDot = trimmed.lastIndexOf('.');
    if (colonIndex > afterLastDot) {
      // Colon is after the last dot, so it's a port separator
      host = trimmed.substring(0, colonIndex);
      const portStr = trimmed.substring(colonIndex + 1);
      port = parseInt(portStr, 10);

      if (!isValidPort(port)) {
        return { valid: false, error: 'Port must be between 1 and 65535' };
      }
    } else {
      host = trimmed;
    }
  } else {
    host = trimmed;
  }

  return { valid: true, host, port };
}

/**
 * Normalize a connection code to a canonical form for SRP authentication.
 *
 * Strips all non-alphanumeric characters (colons, dashes, spaces) and lowercases,
 * producing a canonical representation regardless of input format.
 *
 * Examples:
 *   "94:C6:91:A8:18:EA" → "94c691a818ea"
 *   "94-c6-91-a8-18-ea" → "94c691a818ea"
 *   "94C691A818EA"      → "94c691a818ea"
 *
 * Both the mobile app and the BoardingPass server MUST apply this normalization
 * before SRP key derivation to ensure the same password bytes are used.
 */
export function normalizeConnectionCode(code: string): string {
  return code.replace(/[^a-zA-Z0-9]/g, '').toLowerCase();
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
