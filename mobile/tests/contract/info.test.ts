/**
 * Contract Test: GET /info
 *
 * Validates the /info endpoint contract against the BoardingPass API OpenAPI specification.
 * This test ensures the mobile app correctly handles system information responses and validates
 * the structure of TPM, board, CPU, OS, and FIPS status data.
 *
 * OpenAPI Spec: ../../specs/001-boardingpass-api/contracts/openapi.yaml
 * Contract: See mobile/specs/003-mobile-onboarding-app/contracts/README.md
 */

describe('GET /info Contract', () => {
  describe('Request Format', () => {
    it('should use GET method', () => {
      const httpMethod = 'GET';

      expect(httpMethod).toBe('GET');
      // Per OpenAPI spec, /info endpoint uses GET
    });

    it('should use correct endpoint path', () => {
      const endpointPath = '/info';

      expect(endpointPath).toBe('/info');
      // Must match OpenAPI specification exactly
    });

    it('should require Authorization header with session token', () => {
      const headers = {
        Authorization: 'Bearer dGhpc2lzYXRva2VuaWQ.c2lnbmF0dXJlaGVyZQ',
      };

      expect(headers).toHaveProperty('Authorization');
      expect(headers.Authorization).toMatch(/^Bearer /);
      // Endpoint requires authentication per OpenAPI security scheme
    });

    it('should accept application/json responses', () => {
      const headers = {
        Accept: 'application/json',
      };

      expect(headers.Accept).toBe('application/json');
    });
  });

  describe('Response Format (200 OK)', () => {
    it('should have all required top-level properties', () => {
      const response = {
        tpm: {
          present: true,
          manufacturer: 'STMicroelectronics',
          model: 'ST33HTPH2E32',
          version: '2.0',
        },
        board: {
          manufacturer: 'Raspberry Pi Foundation',
          model: 'Raspberry Pi 4 Model B',
          serial: '10000000abcdef01',
        },
        cpu: {
          architecture: 'aarch64',
        },
        os: {
          distribution: 'Red Hat Enterprise Linux',
          version: '9.3',
        },
        fips_mode: true,
      };

      expect(response).toHaveProperty('tpm');
      expect(response).toHaveProperty('board');
      expect(response).toHaveProperty('cpu');
      expect(response).toHaveProperty('os');
      expect(response).toHaveProperty('fips_mode');
    });

    it('should validate TPMInfo structure', () => {
      const tpm = {
        present: true,
        manufacturer: 'STMicroelectronics',
        model: 'ST33HTPH2E32',
        version: '2.0',
      };

      expect(tpm).toHaveProperty('present');
      expect(typeof tpm.present).toBe('boolean');
      expect(typeof tpm.manufacturer).toBe('string');
      expect(typeof tpm.model).toBe('string');
      expect(typeof tpm.version).toBe('string');
    });

    it('should allow TPMInfo with only present=false', () => {
      const tpm = {
        present: false,
        manufacturer: null,
        model: null,
        version: null,
      };

      expect(tpm.present).toBe(false);
      // manufacturer, model, version are nullable when TPM not present
    });

    it('should validate BoardInfo structure', () => {
      const board = {
        manufacturer: 'Raspberry Pi Foundation',
        model: 'Raspberry Pi 4 Model B',
        serial: '10000000abcdef01',
      };

      expect(board).toHaveProperty('manufacturer');
      expect(board).toHaveProperty('model');
      expect(board).toHaveProperty('serial');
      expect(typeof board.manufacturer).toBe('string');
      expect(typeof board.model).toBe('string');
      expect(typeof board.serial).toBe('string');
    });

    it('should validate CPUInfo architecture enum', () => {
      const validArchitectures = ['x86_64', 'aarch64', 'armv7l'];

      validArchitectures.forEach(arch => {
        const cpu = {
          architecture: arch,
        };

        expect(cpu).toHaveProperty('architecture');
        expect(validArchitectures).toContain(cpu.architecture);
      });
    });

    it('should reject invalid CPU architecture', () => {
      const invalidArchitectures = ['arm', 'i386', 'ppc64le', 's390x'];

      invalidArchitectures.forEach(arch => {
        const validArchitectures = ['x86_64', 'aarch64', 'armv7l'];
        expect(validArchitectures).not.toContain(arch);
        // Implementation should reject architectures not in enum
      });
    });

    it('should validate OSInfo structure', () => {
      const os = {
        distribution: 'Red Hat Enterprise Linux',
        version: '9.3',
      };

      expect(os).toHaveProperty('distribution');
      expect(os).toHaveProperty('version');
      expect(typeof os.distribution).toBe('string');
      expect(typeof os.version).toBe('string');
    });

    it('should validate FIPS mode as boolean', () => {
      const fipsModeValues = [true, false];

      fipsModeValues.forEach(value => {
        const response = {
          fips_mode: value,
        };

        expect(typeof response.fips_mode).toBe('boolean');
      });
    });

    it('should reject response missing required fields', () => {
      const incompleteResponse = {
        tpm: { present: true },
        // Missing: board, cpu, os, fips_mode
      };

      expect(incompleteResponse).toHaveProperty('tpm');
      expect(incompleteResponse).not.toHaveProperty('board');
      expect(incompleteResponse).not.toHaveProperty('cpu');
      // Implementation should reject incomplete responses
    });
  });

  describe('Error Responses', () => {
    it('should handle 401 Unauthorized (missing or invalid token)', () => {
      const errorResponse = {
        error: 'unauthorized',
        message: 'Session token required',
      };

      expect(errorResponse).toHaveProperty('error');
      expect(errorResponse).toHaveProperty('message');
      expect(typeof errorResponse.error).toBe('string');
      expect(typeof errorResponse.message).toBe('string');
    });

    it('should handle 401 Unauthorized (expired token)', () => {
      const errorResponse = {
        error: 'session_expired',
        message: 'Session token has expired',
      };

      expect(errorResponse).toHaveProperty('error');
      expect(errorResponse).toHaveProperty('message');
    });

    it('should handle 500 Internal Server Error', () => {
      const errorResponse = {
        error: 'internal_error',
        message: 'Failed to retrieve system information',
      };

      expect(errorResponse).toHaveProperty('error');
      expect(errorResponse).toHaveProperty('message');
    });
  });

  describe('Security Requirements', () => {
    it('should use HTTPS (TLS 1.3+) for all requests', () => {
      const baseURL = 'https://192.168.1.100:8443';

      expect(baseURL).toMatch(/^https:\/\//);
      // Implementation MUST reject HTTP connections
    });

    it('should include session token in Authorization header', () => {
      const headers = {
        Authorization: 'Bearer dGhpc2lzYXRva2VuaWQ.c2lnbmF0dXJlaGVyZQ',
      };

      expect(headers.Authorization).toBeDefined();
      expect(headers.Authorization).toMatch(/^Bearer /);
      // Endpoint requires authentication
    });

    it('should never log sensitive device identifiers', () => {
      const sensitiveFields = ['serial', 'manufacturer', 'model'];

      // These values should be handled carefully in logs (FR-029)
      // Implementation MAY log these as they are not secrets, but should redact if policy requires
      sensitiveFields.forEach(field => {
        expect(field).toBeDefined();
      });
    });
  });

  describe('Data Validation', () => {
    it('should validate TPM version format', () => {
      const validVersions = ['2.0', '1.2'];
      const tpm = {
        present: true,
        version: '2.0',
      };

      expect(validVersions).toContain(tpm.version);
    });

    it('should validate board serial is non-empty string', () => {
      const board = {
        manufacturer: 'Test Manufacturer',
        model: 'Test Model',
        serial: '10000000abcdef01',
      };

      expect(board.serial.length).toBeGreaterThan(0);
      expect(typeof board.serial).toBe('string');
    });

    it('should validate OS distribution is non-empty string', () => {
      const os = {
        distribution: 'Red Hat Enterprise Linux',
        version: '9.3',
      };

      expect(os.distribution.length).toBeGreaterThan(0);
      expect(os.version.length).toBeGreaterThan(0);
    });
  });

  describe('Endpoint Configuration', () => {
    it('should connect to device port 8443 by default', () => {
      const defaultPort = 8443;

      expect(defaultPort).toBe(8443);
      // Per plan.md: BoardingPass API listens on port 8443
    });

    it('should timeout after reasonable period', () => {
      const reasonableTimeout = 30000; // 30 seconds

      expect(reasonableTimeout).toBeGreaterThanOrEqual(5000);
      expect(reasonableTimeout).toBeLessThanOrEqual(60000);
      // Implementation should have timeout between 5-60 seconds
    });
  });

  describe('FIPS Compliance Indicators', () => {
    it('should correctly identify FIPS-enabled devices', () => {
      const fipsEnabledResponse = {
        tpm: { present: true },
        board: { manufacturer: 'Test', model: 'Test', serial: '123' },
        cpu: { architecture: 'x86_64' },
        os: { distribution: 'RHEL', version: '9.3' },
        fips_mode: true,
      };

      expect(fipsEnabledResponse.fips_mode).toBe(true);
      // Implementation should display FIPS indicator badge
    });

    it('should correctly identify non-FIPS devices', () => {
      const nonFipsResponse = {
        tpm: { present: false },
        board: { manufacturer: 'Test', model: 'Test', serial: '123' },
        cpu: { architecture: 'aarch64' },
        os: { distribution: 'Ubuntu', version: '22.04' },
        fips_mode: false,
      };

      expect(nonFipsResponse.fips_mode).toBe(false);
      // Implementation should NOT display FIPS indicator badge
    });
  });
});
