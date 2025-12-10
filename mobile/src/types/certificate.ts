/**
 * Certificate Entity Types
 *
 * TLS certificate information and pinning data for device connections.
 * Implements Trust-On-First-Use (TOFU) with certificate pinning.
 */

export type TrustStatus = 'trusted_ca' | 'self_signed_trusted' | 'self_signed_new' | 'changed';

/**
 * CertificateInfo
 *
 * Certificate information and pinning data for secure device connections.
 */
export interface CertificateInfo {
  // Device association
  deviceId: string;

  // Certificate identity
  fingerprint: string; // SHA-256 fingerprint (64-char hex)

  // Certificate details
  subject: string; // X.509 subject (e.g., "CN=device.local")
  issuer: string; // X.509 issuer
  validFrom: Date;
  validTo: Date;

  // Certificate type
  isSelfSigned: boolean;
  issuedByTrustedCA: boolean;

  // Trust status
  trustStatus: TrustStatus;

  // Pinning metadata
  pinnedAt: Date;
  lastVerified: Date;

  // User decision
  userConfirmedAt?: Date; // For self-signed certs
}

/**
 * Certificate validation
 */
export const isCertificateValid = (cert: CertificateInfo): boolean => {
  const now = new Date();
  return now >= new Date(cert.validFrom) && now <= new Date(cert.validTo);
};

export const isCertificateTrusted = (cert: CertificateInfo): boolean => {
  return cert.trustStatus === 'trusted_ca' || cert.trustStatus === 'self_signed_trusted';
};

export const isCertificateChanged = (cert: CertificateInfo): boolean => {
  return cert.trustStatus === 'changed';
};

export const needsUserTrust = (cert: CertificateInfo): boolean => {
  return cert.trustStatus === 'self_signed_new' || cert.trustStatus === 'changed';
};

/**
 * Fingerprint validation
 */
export const isValidFingerprint = (fingerprint: string): boolean => {
  // SHA-256 fingerprint is 64 hex characters
  const fingerprintPattern = /^[a-fA-F0-9]{64}$/;
  return fingerprintPattern.test(fingerprint);
};

/**
 * Get human-readable trust status
 */
export const getTrustStatusDescription = (status: TrustStatus): string => {
  switch (status) {
    case 'trusted_ca':
      return 'Certificate issued by trusted authority';
    case 'self_signed_trusted':
      return 'Self-signed certificate (previously trusted)';
    case 'self_signed_new':
      return 'New self-signed certificate';
    case 'changed':
      return 'Certificate has changed - verification required';
  }
};

/**
 * Get trust status icon/color
 */
export const getTrustStatusIndicator = (status: TrustStatus): { icon: string; color: string } => {
  switch (status) {
    case 'trusted_ca':
      return { icon: 'check-circle', color: '#4CAF50' }; // Green
    case 'self_signed_trusted':
      return { icon: 'shield-check', color: '#FFC107' }; // Yellow
    case 'self_signed_new':
      return { icon: 'alert', color: '#FF9800' }; // Orange
    case 'changed':
      return { icon: 'alert-circle', color: '#F44336' }; // Red
  }
};
