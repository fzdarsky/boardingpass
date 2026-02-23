/**
 * Certificate Validation Service
 *
 * Implements certificate pinning with Trust-On-First-Use (TOFU) pattern.
 * Validates certificates on subsequent connections and detects changes.
 */

import { CertificateInfo } from '../../types/certificate';
import { useSecureStorage, STORAGE_KEYS } from '../../hooks/useSecureStorage';
import {
  fetchCertificate,
  parseCertificate,
  validateCertificatePin,
  isCertificateExpired,
  fetchCertificateMock,
} from './utils';

export interface CertificateValidationResult {
  certificate: CertificateInfo;
  isValid: boolean;
  requiresUserTrust: boolean;
  error?: string;
}

/**
 * Certificate Validation Service
 *
 * Manages certificate pinning and validation for device connections.
 */
export class CertificateValidationService {
  constructor(
    private storage: ReturnType<typeof useSecureStorage>
  ) {}

  /**
   * Validate certificate for device connection
   *
   * @param deviceId - Device identifier
   * @param host - Device hostname or IP
   * @param port - Device port (default 9443)
   * @param useMock - Use mock certificate for testing (default false)
   * @returns Validation result with certificate info
   */
  async validateCertificate(
    deviceId: string,
    host: string,
    port: number = 9443,
    useMock: boolean = false
  ): Promise<CertificateValidationResult> {
    try {
      // Fetch current certificate from device
      const rawCert = useMock
        ? await fetchCertificateMock(host, port)
        : await fetchCertificate(host, port);

      // Load existing pinned certificate (if any)
      const pinnedCert = await this.loadPinnedCertificate(deviceId);

      // Parse certificate and determine trust status
      const currentCert = await parseCertificate(deviceId, rawCert, pinnedCert || undefined);

      // Check if certificate is expired
      if (isCertificateExpired(currentCert)) {
        return {
          certificate: currentCert,
          isValid: false,
          requiresUserTrust: false,
          error: 'Certificate has expired',
        };
      }

      // If no pinned cert, this is first connection
      if (!pinnedCert) {
        return this.handleFirstConnection(currentCert);
      }

      // Validate pinned certificate
      return this.handleSubsequentConnection(currentCert, pinnedCert);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      throw new Error(`Certificate validation failed: ${errorMessage}`);
    }
  }

  /**
   * Handle first connection (TOFU - Trust On First Use)
   */
  private handleFirstConnection(certificate: CertificateInfo): CertificateValidationResult {
    if (certificate.trustStatus === 'trusted_ca') {
      // CA-signed certificate - trust immediately
      return {
        certificate,
        isValid: true,
        requiresUserTrust: false,
      };
    } else if (certificate.trustStatus === 'self_signed_new') {
      // Self-signed certificate - require user confirmation
      return {
        certificate,
        isValid: false,
        requiresUserTrust: true,
      };
    }

    // Shouldn't reach here, but handle gracefully
    return {
      certificate,
      isValid: false,
      requiresUserTrust: true,
    };
  }

  /**
   * Handle subsequent connection (validate pin)
   */
  private handleSubsequentConnection(
    current: CertificateInfo,
    pinned: CertificateInfo
  ): CertificateValidationResult {
    const pinValid = validateCertificatePin(current, pinned);

    if (!pinValid) {
      // Certificate changed! Require user confirmation
      return {
        certificate: current,
        isValid: false,
        requiresUserTrust: true,
        error: 'Certificate has changed from previously pinned certificate',
      };
    }

    // Pin matches - connection is valid
    return {
      certificate: current,
      isValid: true,
      requiresUserTrust: false,
    };
  }

  /**
   * Pin certificate after user confirmation
   *
   * @param certificate - Certificate to pin
   */
  async pinCertificate(certificate: CertificateInfo): Promise<void> {
    try {
      // Update trust status if self-signed
      const pinnedCert: CertificateInfo = {
        ...certificate,
        trustStatus:
          certificate.isSelfSigned && certificate.trustStatus === 'self_signed_new'
            ? 'self_signed_trusted'
            : certificate.trustStatus,
        pinnedAt: certificate.pinnedAt || new Date(),
        userConfirmedAt: certificate.isSelfSigned ? new Date() : certificate.userConfirmedAt,
      };

      // Store in secure storage
      const storageKey = STORAGE_KEYS.certificatePin(certificate.deviceId);
      await this.storage.saveJSON(storageKey, this.serializeCertificate(pinnedCert));
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      throw new Error(`Failed to pin certificate: ${errorMessage}`);
    }
  }

  /**
   * Load pinned certificate from secure storage
   *
   * @param deviceId - Device identifier
   * @returns Pinned certificate or null if not found
   */
  async loadPinnedCertificate(deviceId: string): Promise<CertificateInfo | null> {
    try {
      const storageKey = STORAGE_KEYS.certificatePin(deviceId);
      const stored = await this.storage.getJSON<SerializedCertificate>(storageKey);

      if (!stored) {
        return null;
      }

      return this.deserializeCertificate(stored);
    } catch (error) {
      // If storage fails, treat as no pinned cert
      console.warn(`Failed to load pinned certificate for ${deviceId}:`, error);
      return null;
    }
  }

  /**
   * Remove pinned certificate
   *
   * @param deviceId - Device identifier
   */
  async removePinnedCertificate(deviceId: string): Promise<void> {
    try {
      const storageKey = STORAGE_KEYS.certificatePin(deviceId);
      await this.storage.deleteItem(storageKey);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      throw new Error(`Failed to remove pinned certificate: ${errorMessage}`);
    }
  }

  /**
   * Check if device has pinned certificate
   *
   * @param deviceId - Device identifier
   * @returns True if certificate is pinned
   */
  async hasPinnedCertificate(deviceId: string): Promise<boolean> {
    const storageKey = STORAGE_KEYS.certificatePin(deviceId);
    return await this.storage.hasItem(storageKey);
  }

  /**
   * Serialize certificate for storage (convert Dates to ISO strings)
   */
  private serializeCertificate(cert: CertificateInfo): SerializedCertificate {
    return {
      ...cert,
      validFrom: cert.validFrom.toISOString(),
      validTo: cert.validTo.toISOString(),
      pinnedAt: cert.pinnedAt.toISOString(),
      lastVerified: cert.lastVerified.toISOString(),
      userConfirmedAt: cert.userConfirmedAt?.toISOString(),
    };
  }

  /**
   * Deserialize certificate from storage (convert ISO strings to Dates)
   */
  private deserializeCertificate(stored: SerializedCertificate): CertificateInfo {
    return {
      ...stored,
      validFrom: new Date(stored.validFrom),
      validTo: new Date(stored.validTo),
      pinnedAt: new Date(stored.pinnedAt),
      lastVerified: new Date(stored.lastVerified),
      userConfirmedAt: stored.userConfirmedAt ? new Date(stored.userConfirmedAt) : undefined,
    };
  }
}

/**
 * Serialized certificate for storage (Dates as ISO strings)
 */
interface SerializedCertificate {
  deviceId: string;
  fingerprint: string;
  subject: string;
  issuer: string;
  validFrom: string; // ISO date string
  validTo: string; // ISO date string
  isSelfSigned: boolean;
  issuedByTrustedCA: boolean;
  trustStatus: 'trusted_ca' | 'self_signed_trusted' | 'self_signed_new' | 'changed';
  pinnedAt: string; // ISO date string
  lastVerified: string; // ISO date string
  userConfirmedAt?: string; // ISO date string
}

/**
 * Create certificate validation service instance
 *
 * Note: This function should be called from within a React component
 * so it has access to the useSecureStorage hook.
 *
 * @param storage - Secure storage instance from useSecureStorage hook
 */
export function createCertificateValidationService(
  storage: ReturnType<typeof useSecureStorage>
): CertificateValidationService {
  return new CertificateValidationService(storage);
}
