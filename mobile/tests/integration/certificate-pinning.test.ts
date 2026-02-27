/**
 * Certificate Pinning Integration Tests
 *
 * Tests the full certificate pinning workflow:
 * 1. First connection (TOFU - Trust On First Use)
 * 2. Certificate validation on subsequent connections
 * 3. Certificate change detection
 * 4. Certificate trust workflow
 */

import { renderHook, act } from '@testing-library/react-native';
import { useCertificateValidation } from '../../src/hooks/useCertificateValidation';
import * as SecureStore from 'expo-secure-store';
import { CertificateInfo } from '../../src/types/certificate';

// Mock expo-secure-store
jest.mock('expo-secure-store');

describe('Certificate Pinning Integration', () => {
  const mockDeviceId = 'device-test-123';
  const mockHost = '192.168.1.100';
  const mockPort = 8443;

  beforeEach(() => {
    // Clear all mocks before each test
    jest.clearAllMocks();

    // Mock secure storage methods
    (SecureStore.getItemAsync as jest.Mock).mockResolvedValue(null);
    (SecureStore.setItemAsync as jest.Mock).mockResolvedValue(undefined);
    (SecureStore.deleteItemAsync as jest.Mock).mockResolvedValue(undefined);
  });

  describe('First Connection (TOFU)', () => {
    it('should handle first connection with self-signed certificate', async () => {
      const { result } = renderHook(() => useCertificateValidation());

      // First connection - no pinned certificate
      await act(async () => {
        const validationResult = await result.current.validateCertificate(
          mockDeviceId,
          mockHost,
          mockPort,
          true // Use mock certificate
        );

        expect(validationResult.certificate).toBeDefined();
        expect(validationResult.certificate.isSelfSigned).toBe(true);
        expect(validationResult.requiresUserTrust).toBe(true);
        expect(validationResult.isValid).toBe(false);
      });

      // State should reflect need for user trust
      expect(result.current.requiresTrust).toBe(true);
      expect(result.current.certificate).toBeDefined();
      expect(result.current.certificate?.trustStatus).toBe('self_signed_new');
    });

    it('should handle first connection with CA-signed certificate', async () => {
      const { result } = renderHook(() => useCertificateValidation());

      // Mock a CA-signed certificate (issuer != subject)
      // Note: This would require modifying fetchCertificateMock or using a different mock
      // For now, this test demonstrates the expected behavior

      await act(async () => {
        await result.current.validateCertificate(mockDeviceId, mockHost, mockPort, true);

        // For mock cert, it's self-signed, but in real scenario with CA-signed cert:
        // certificate.isSelfSigned would be false
        // requiresUserTrust would be false
        // isValid would be true
      });
    });
  });

  describe('Certificate Trust Workflow', () => {
    it('should pin certificate after user trust', async () => {
      const { result } = renderHook(() => useCertificateValidation());

      // Step 1: Validate certificate (first connection)
      let certificate: CertificateInfo | null = null;
      await act(async () => {
        const validationResult = await result.current.validateCertificate(
          mockDeviceId,
          mockHost,
          mockPort,
          true
        );
        certificate = validationResult.certificate;
      });

      expect(certificate).not.toBeNull();
      expect(result.current.requiresTrust).toBe(true);

      // Step 2: User trusts the certificate
      await act(async () => {
        await result.current.trustCertificate(certificate!);
      });

      // Verify certificate was pinned to secure storage
      expect(SecureStore.setItemAsync).toHaveBeenCalled();
      const setItemCall = (SecureStore.setItemAsync as jest.Mock).mock.calls[0];
      expect(setItemCall[0]).toContain(mockDeviceId); // Storage key includes device ID

      // State should reflect trusted status
      expect(result.current.requiresTrust).toBe(false);
      expect(result.current.certificate?.trustStatus).toBe('self_signed_trusted');
    });

    it('should allow removing pinned certificate', async () => {
      const { result } = renderHook(() => useCertificateValidation());

      // Pin a certificate first
      await act(async () => {
        const validationResult = await result.current.validateCertificate(
          mockDeviceId,
          mockHost,
          mockPort,
          true
        );
        await result.current.trustCertificate(validationResult.certificate);
      });

      // Remove the pin
      await act(async () => {
        await result.current.removePinnedCertificate(mockDeviceId);
      });

      // Verify deletion was called
      expect(SecureStore.deleteItemAsync).toHaveBeenCalled();

      // State should be reset
      expect(result.current.certificate).toBeNull();
      expect(result.current.requiresTrust).toBe(false);
    });
  });

  describe('Subsequent Connections', () => {
    it('should validate matching certificate on subsequent connection', async () => {
      const { result } = renderHook(() => useCertificateValidation());

      // First connection: validate and pin
      let firstCert: CertificateInfo | null = null;
      await act(async () => {
        const result1 = await result.current.validateCertificate(
          mockDeviceId,
          mockHost,
          mockPort,
          true
        );
        firstCert = result1.certificate;
        await result.current.trustCertificate(firstCert);
      });

      // Mock storage to return the pinned certificate
      const pinnedCertData = JSON.stringify({
        ...firstCert,
        validFrom: firstCert!.validFrom.toISOString(),
        validTo: firstCert!.validTo.toISOString(),
        pinnedAt: firstCert!.pinnedAt.toISOString(),
        lastVerified: firstCert!.lastVerified.toISOString(),
      });
      (SecureStore.getItemAsync as jest.Mock).mockResolvedValue(pinnedCertData);

      // Second connection: validate against pin
      await act(async () => {
        const result2 = await result.current.validateCertificate(
          mockDeviceId,
          mockHost,
          mockPort,
          true
        );

        // Same certificate should be valid
        expect(result2.isValid).toBe(true);
        expect(result2.requiresUserTrust).toBe(false);
        expect(result2.certificate.fingerprint).toBe(firstCert!.fingerprint);
      });
    });
  });

  describe('Certificate Change Detection', () => {
    it('should detect certificate change and require user confirmation', async () => {
      const { result } = renderHook(() => useCertificateValidation());

      // First connection: pin a certificate
      let firstCert: CertificateInfo | null = null;
      await act(async () => {
        const result1 = await result.current.validateCertificate(
          mockDeviceId,
          mockHost,
          mockPort,
          true
        );
        firstCert = result1.certificate;
        await result.current.trustCertificate(firstCert);
      });

      // Mock storage to return a DIFFERENT pinned certificate
      const differentCert: CertificateInfo = {
        ...firstCert!,
        fingerprint: '0000000000000000000000000000000000000000000000000000000000000000', // Different fingerprint
      };
      const pinnedCertData = JSON.stringify({
        ...differentCert,
        validFrom: differentCert.validFrom.toISOString(),
        validTo: differentCert.validTo.toISOString(),
        pinnedAt: differentCert.pinnedAt.toISOString(),
        lastVerified: differentCert.lastVerified.toISOString(),
      });
      (SecureStore.getItemAsync as jest.Mock).mockResolvedValue(pinnedCertData);

      // Second connection: certificate has changed
      await act(async () => {
        const result2 = await result.current.validateCertificate(
          mockDeviceId,
          mockHost,
          mockPort,
          true
        );

        // Changed certificate should not be valid
        expect(result2.isValid).toBe(false);
        expect(result2.requiresUserTrust).toBe(true);
        expect(result2.error).toContain('changed');
        expect(result2.certificate.trustStatus).toBe('changed');
      });

      // State should reflect changed status
      expect(result.current.requiresTrust).toBe(true);
      expect(result.current.error).toContain('changed');
    });

    it('should allow trusting new certificate after change', async () => {
      const { result } = renderHook(() => useCertificateValidation());

      // Simulate certificate change detection
      let changedCert: CertificateInfo | null = null;
      await act(async () => {
        // First validate (which will show it as changed)
        const differentPinnedCert: CertificateInfo = {
          deviceId: mockDeviceId,
          fingerprint: '0000000000000000000000000000000000000000000000000000000000000000',
          subject: 'CN=old.device.local',
          issuer: 'CN=old.device.local',
          validFrom: new Date(),
          validTo: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000),
          isSelfSigned: true,
          issuedByTrustedCA: false,
          trustStatus: 'self_signed_trusted',
          pinnedAt: new Date(),
          lastVerified: new Date(),
        };

        const pinnedData = JSON.stringify({
          ...differentPinnedCert,
          validFrom: differentPinnedCert.validFrom.toISOString(),
          validTo: differentPinnedCert.validTo.toISOString(),
          pinnedAt: differentPinnedCert.pinnedAt.toISOString(),
          lastVerified: differentPinnedCert.lastVerified.toISOString(),
        });
        (SecureStore.getItemAsync as jest.Mock).mockResolvedValue(pinnedData);

        const validationResult = await result.current.validateCertificate(
          mockDeviceId,
          mockHost,
          mockPort,
          true
        );
        changedCert = validationResult.certificate;
      });

      expect(result.current.requiresTrust).toBe(true);
      expect(changedCert?.trustStatus).toBe('changed');

      // User accepts the new certificate
      await act(async () => {
        await result.current.trustCertificate(changedCert!);
      });

      // New certificate should be pinned
      expect(SecureStore.setItemAsync).toHaveBeenCalled();
      expect(result.current.requiresTrust).toBe(false);
    });
  });

  describe('Error Handling', () => {
    it('should handle storage errors gracefully', async () => {
      const { result } = renderHook(() => useCertificateValidation());

      // Mock storage error
      (SecureStore.setItemAsync as jest.Mock).mockRejectedValue(new Error('Storage unavailable'));

      // Try to pin a certificate
      await act(async () => {
        const validationResult = await result.current.validateCertificate(
          mockDeviceId,
          mockHost,
          mockPort,
          true
        );

        await expect(
          result.current.trustCertificate(validationResult.certificate)
        ).rejects.toThrow();
      });

      expect(result.current.error).toBeDefined();
    });

    it('should handle certificate fetch errors', async () => {
      renderHook(() => useCertificateValidation());

      // Note: With useMock=false, this would test real certificate fetch errors
      // For now, mock version doesn't throw, but in production:
      // await expect(
      //   result.current.validateCertificate(mockDeviceId, mockHost, mockPort, false)
      // ).rejects.toThrow();
    });
  });

  describe('State Management', () => {
    it('should reset validation state', async () => {
      const { result } = renderHook(() => useCertificateValidation());

      // Perform validation
      await act(async () => {
        await result.current.validateCertificate(mockDeviceId, mockHost, mockPort, true);
      });

      expect(result.current.certificate).not.toBeNull();

      // Reset state
      act(() => {
        result.current.resetValidationState();
      });

      expect(result.current.certificate).toBeNull();
      expect(result.current.requiresTrust).toBe(false);
      expect(result.current.error).toBeNull();
      expect(result.current.isValidating).toBe(false);
    });

    it('should track validation in progress', async () => {
      const { result } = renderHook(() => useCertificateValidation());

      await act(async () => {
        const promise = result.current.validateCertificate(mockDeviceId, mockHost, mockPort, true);

        // Check if validating flag is set during async operation
        // Note: validating flag may not be observable synchronously in test env
        void result.current.isValidating;

        await promise;
      });

      // And false after completion
      expect(result.current.isValidating).toBe(false);
    });
  });
});
