/**
 * Certificate Validation Hook
 *
 * React hook for certificate validation and pinning workflow.
 * Integrates with API client for secure HTTPS connections.
 */

import { useState, useCallback } from 'react';
import { useSecureStorage } from './useSecureStorage';
import { CertificateInfo } from '../types/certificate';
import {
  createCertificateValidationService,
  CertificateValidationResult,
} from '../services/certificates/validation';

export interface CertificateValidationState {
  isValidating: boolean;
  certificate: CertificateInfo | null;
  requiresTrust: boolean;
  error: string | null;
}

/**
 * useCertificateValidation
 *
 * Hook for managing certificate validation and trust workflow:
 * 1. Validate certificate on connection
 * 2. Show trust dialog if needed (self-signed or changed)
 * 3. Pin certificate after user confirmation
 * 4. Validate pinned certificate on subsequent connections
 */
export function useCertificateValidation() {
  const storage = useSecureStorage();
  const [validationService] = useState(() => createCertificateValidationService(storage));

  const [state, setState] = useState<CertificateValidationState>({
    isValidating: false,
    certificate: null,
    requiresTrust: false,
    error: null,
  });

  /**
   * Validate certificate for device connection
   *
   * @param deviceId - Device identifier
   * @param host - Device hostname or IP
   * @param port - Device port (default 8443)
   * @param useMock - Use mock certificate for testing
   * @returns Validation result
   */
  const validateCertificate = useCallback(
    async (
      deviceId: string,
      host: string,
      port: number = 8443,
      useMock: boolean = true // Set to true for development until native cert fetching is implemented
    ): Promise<CertificateValidationResult> => {
      setState(prev => ({ ...prev, isValidating: true, error: null }));

      try {
        const result = await validationService.validateCertificate(deviceId, host, port, useMock);

        setState({
          isValidating: false,
          certificate: result.certificate,
          requiresTrust: result.requiresUserTrust,
          error: result.error || null,
        });

        return result;
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : 'Certificate validation failed';
        setState({
          isValidating: false,
          certificate: null,
          requiresTrust: false,
          error: errorMessage,
        });
        throw error;
      }
    },
    [validationService]
  );

  /**
   * Trust and pin certificate after user confirmation
   *
   * @param certificate - Certificate to trust
   */
  const trustCertificate = useCallback(
    async (certificate: CertificateInfo): Promise<void> => {
      try {
        await validationService.pinCertificate(certificate);

        setState(prev => ({
          ...prev,
          certificate: { ...certificate, trustStatus: 'self_signed_trusted' },
          requiresTrust: false,
        }));
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Failed to trust certificate';
        setState(prev => ({ ...prev, error: errorMessage }));
        throw error;
      }
    },
    [validationService]
  );

  /**
   * Load pinned certificate for device
   *
   * @param deviceId - Device identifier
   * @returns Pinned certificate or null
   */
  const loadPinnedCertificate = useCallback(
    async (deviceId: string): Promise<CertificateInfo | null> => {
      try {
        return await validationService.loadPinnedCertificate(deviceId);
      } catch (error) {
        console.warn('Failed to load pinned certificate:', error);
        return null;
      }
    },
    [validationService]
  );

  /**
   * Remove pinned certificate
   *
   * @param deviceId - Device identifier
   */
  const removePinnedCertificate = useCallback(
    async (deviceId: string): Promise<void> => {
      try {
        await validationService.removePinnedCertificate(deviceId);
        setState(prev => ({ ...prev, certificate: null, requiresTrust: false }));
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : 'Failed to remove certificate';
        setState(prev => ({ ...prev, error: errorMessage }));
        throw error;
      }
    },
    [validationService]
  );

  /**
   * Check if device has pinned certificate
   *
   * @param deviceId - Device identifier
   */
  const hasPinnedCertificate = useCallback(
    async (deviceId: string): Promise<boolean> => {
      return await validationService.hasPinnedCertificate(deviceId);
    },
    [validationService]
  );

  /**
   * Reset validation state
   */
  const resetValidationState = useCallback(() => {
    setState({
      isValidating: false,
      certificate: null,
      requiresTrust: false,
      error: null,
    });
  }, []);

  return {
    ...state,
    validateCertificate,
    trustCertificate,
    loadPinnedCertificate,
    removePinnedCertificate,
    hasPinnedCertificate,
    resetValidationState,
  };
}
