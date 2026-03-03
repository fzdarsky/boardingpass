/**
 * Network Validation Utilities
 *
 * Validators for network configuration inputs used in the enrollment wizard.
 * Each returns an error message string or null if valid.
 */

/**
 * Validate hostname per RFC 1123.
 * - 1-253 characters total
 * - Labels separated by dots, each 1-63 characters
 * - Alphanumeric + hyphens, no leading/trailing hyphens per label
 */
export function validateHostname(hostname: string): string | null {
  if (!hostname) {
    return 'Hostname is required';
  }
  if (hostname.length > 253) {
    return 'Hostname must be at most 253 characters';
  }

  const labels = hostname.split('.');
  for (const label of labels) {
    if (label.length === 0) {
      return 'Hostname labels must not be empty';
    }
    if (label.length > 63) {
      return 'Each hostname label must be at most 63 characters';
    }
    if (!/^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$/.test(label)) {
      return 'Hostname labels must be alphanumeric with optional hyphens (no leading/trailing hyphens)';
    }
  }

  return null;
}

/**
 * Validate an IPv4 address (dotted-decimal notation).
 */
export function validateIPv4(ip: string): string | null {
  if (!ip) {
    return 'IPv4 address is required';
  }

  const parts = ip.split('.');
  if (parts.length !== 4) {
    return 'IPv4 address must have exactly 4 octets';
  }

  for (const part of parts) {
    const num = parseInt(part, 10);
    if (isNaN(num) || num < 0 || num > 255 || String(num) !== part) {
      return 'Each IPv4 octet must be a number between 0 and 255';
    }
  }

  return null;
}

/**
 * Validate an IPv6 address.
 * Supports full and abbreviated (::) notation.
 * Optionally allows prefix length suffix (e.g., "2001:db8::1/64").
 */
export function validateIPv6(ip: string, allowPrefix = false): string | null {
  if (!ip) {
    return 'IPv6 address is required';
  }

  let addr = ip;
  if (allowPrefix && ip.includes('/')) {
    const [addrPart, prefixPart] = ip.split('/');
    const prefix = parseInt(prefixPart, 10);
    if (isNaN(prefix) || prefix < 0 || prefix > 128 || String(prefix) !== prefixPart) {
      return 'IPv6 prefix length must be between 0 and 128';
    }
    addr = addrPart;
  } else if (!allowPrefix && ip.includes('/')) {
    return 'IPv6 address must not include prefix length';
  }

  // Check for :: (at most one occurrence)
  const doubleColonCount = (addr.match(/::/g) || []).length;
  if (doubleColonCount > 1) {
    return 'IPv6 address may contain at most one ::';
  }

  // Split and validate groups
  if (doubleColonCount === 1) {
    const [left, right] = addr.split('::');
    const leftGroups = left ? left.split(':') : [];
    const rightGroups = right ? right.split(':') : [];
    const totalGroups = leftGroups.length + rightGroups.length;

    if (totalGroups > 7) {
      return 'Invalid IPv6 address';
    }

    for (const group of [...leftGroups, ...rightGroups]) {
      if (!isValidIPv6Group(group)) {
        return 'Invalid IPv6 address group';
      }
    }
  } else {
    const groups = addr.split(':');
    if (groups.length !== 8) {
      return 'IPv6 address must have exactly 8 groups (or use :: abbreviation)';
    }
    for (const group of groups) {
      if (!isValidIPv6Group(group)) {
        return 'Invalid IPv6 address group';
      }
    }
  }

  return null;
}

function isValidIPv6Group(group: string): boolean {
  return /^[0-9a-fA-F]{1,4}$/.test(group);
}

/**
 * Validate a subnet mask in dotted-decimal notation.
 * Must be a valid contiguous mask (e.g., 255.255.255.0).
 */
export function validateSubnetMask(mask: string): string | null {
  if (!mask) {
    return 'Subnet mask is required';
  }

  // Support CIDR prefix notation (e.g., "24")
  const cidr = parseInt(mask, 10);
  if (String(cidr) === mask && cidr >= 0 && cidr <= 32) {
    return null;
  }

  // Dotted-decimal
  const parts = mask.split('.');
  if (parts.length !== 4) {
    return 'Subnet mask must have exactly 4 octets or be a CIDR prefix (0-32)';
  }

  let binary = '';
  for (const part of parts) {
    const num = parseInt(part, 10);
    if (isNaN(num) || num < 0 || num > 255 || String(num) !== part) {
      return 'Each subnet mask octet must be a number between 0 and 255';
    }
    binary += num.toString(2).padStart(8, '0');
  }

  // Must be contiguous 1s followed by contiguous 0s
  if (!/^1*0*$/.test(binary)) {
    return 'Subnet mask must be contiguous (e.g., 255.255.255.0)';
  }

  return null;
}

/**
 * Validate that a gateway IP is within the subnet defined by address + mask.
 */
export function validateGatewayInSubnet(
  gateway: string,
  address: string,
  subnetMask: string
): string | null {
  const gwErr = validateIPv4(gateway);
  if (gwErr) return `Gateway: ${gwErr}`;

  const addrErr = validateIPv4(address);
  if (addrErr) return null; // Can't validate without a valid address

  const maskErr = validateSubnetMask(subnetMask);
  if (maskErr) return null; // Can't validate without a valid mask

  const prefix = maskToPrefix(subnetMask);
  const gwNum = ipToNum(gateway);
  const addrNum = ipToNum(address);
  // eslint-disable-next-line no-bitwise
  const maskNum = (0xffffffff << (32 - prefix)) >>> 0;

  // eslint-disable-next-line no-bitwise
  if ((gwNum & maskNum) !== (addrNum & maskNum)) {
    return 'Gateway must be in the same subnet as the address';
  }

  return null;
}

/**
 * Validate a port number (1-65535).
 */
export function validatePort(port: number | string): string | null {
  const num = typeof port === 'string' ? parseInt(port, 10) : port;
  if (isNaN(num) || !Number.isInteger(num) || num < 1 || num > 65535) {
    return 'Port must be a number between 1 and 65535';
  }
  return null;
}

/**
 * Validate an HTTPS URL.
 */
export function validateHttpsUrl(url: string): string | null {
  if (!url) {
    return 'URL is required';
  }

  try {
    const parsed = new URL(url);
    if (parsed.protocol !== 'https:') {
      return 'URL must use HTTPS';
    }
  } catch {
    return 'Invalid URL format';
  }

  return null;
}

/**
 * Validate an NTP server hostname or IP address.
 */
export function validateNtpServer(server: string): string | null {
  if (!server) {
    return 'NTP server is required';
  }

  // Accept valid IPv4
  if (validateIPv4(server) === null) {
    return null;
  }

  // Accept valid hostname
  if (validateHostname(server) === null) {
    return null;
  }

  return 'NTP server must be a valid hostname or IPv4 address';
}

// ── Helpers ──

function ipToNum(ip: string): number {
  const parts = ip.split('.').map(Number);
  // eslint-disable-next-line no-bitwise
  return ((parts[0] << 24) | (parts[1] << 16) | (parts[2] << 8) | parts[3]) >>> 0;
}

function maskToPrefix(mask: string): number {
  const cidr = parseInt(mask, 10);
  if (String(cidr) === mask && cidr >= 0 && cidr <= 32) {
    return cidr;
  }
  const parts = mask.split('.').map(Number);
  let binary = '';
  for (const p of parts) {
    binary += p.toString(2).padStart(8, '0');
  }
  return binary.indexOf('0') === -1 ? 32 : binary.indexOf('0');
}
