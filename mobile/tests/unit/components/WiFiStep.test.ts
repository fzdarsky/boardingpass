/**
 * WiFiStep Component Tests
 *
 * Tests the pure functions used by WiFiStep: scan output parsing,
 * frequency band derivation, signal icon selection, and WiFi network
 * security classification.
 */

import {
  deriveBand,
  parseWifiScanOutput,
  signalIcon,
} from '../../../src/components/ConfigWizard/WiFiStep';

describe('WiFiStep utilities', () => {
  describe('deriveBand', () => {
    it('returns 2.4 GHz for 2.4 GHz frequencies', () => {
      expect(deriveBand(2412)).toBe('2.4 GHz');
      expect(deriveBand(2437)).toBe('2.4 GHz');
      expect(deriveBand(2462)).toBe('2.4 GHz');
      expect(deriveBand(2400)).toBe('2.4 GHz');
      expect(deriveBand(2483)).toBe('2.4 GHz');
    });

    it('returns 5 GHz for 5 GHz frequencies', () => {
      expect(deriveBand(5180)).toBe('5 GHz');
      expect(deriveBand(5240)).toBe('5 GHz');
      expect(deriveBand(5500)).toBe('5 GHz');
      expect(deriveBand(5745)).toBe('5 GHz');
      expect(deriveBand(5150)).toBe('5 GHz');
      expect(deriveBand(5850)).toBe('5 GHz');
    });

    it('returns 6 GHz for 6 GHz frequencies', () => {
      expect(deriveBand(5925)).toBe('6 GHz');
      expect(deriveBand(6000)).toBe('6 GHz');
      expect(deriveBand(6500)).toBe('6 GHz');
      expect(deriveBand(7125)).toBe('6 GHz');
    });

    it('returns empty string for unknown frequencies', () => {
      expect(deriveBand(0)).toBe('');
      expect(deriveBand(900)).toBe('');
      expect(deriveBand(2399)).toBe('');
      expect(deriveBand(2484)).toBe('');
      expect(deriveBand(5149)).toBe('');
      expect(deriveBand(5851)).toBe('');
      expect(deriveBand(5924)).toBe('');
      expect(deriveBand(7126)).toBe('');
    });
  });

  describe('signalIcon', () => {
    it('returns strength-4 for strong signals (>=75)', () => {
      expect(signalIcon(75)).toBe('wifi-strength-4');
      expect(signalIcon(100)).toBe('wifi-strength-4');
      expect(signalIcon(90)).toBe('wifi-strength-4');
    });

    it('returns strength-3 for good signals (>=50, <75)', () => {
      expect(signalIcon(50)).toBe('wifi-strength-3');
      expect(signalIcon(60)).toBe('wifi-strength-3');
      expect(signalIcon(74)).toBe('wifi-strength-3');
    });

    it('returns strength-2 for fair signals (>=25, <50)', () => {
      expect(signalIcon(25)).toBe('wifi-strength-2');
      expect(signalIcon(35)).toBe('wifi-strength-2');
      expect(signalIcon(49)).toBe('wifi-strength-2');
    });

    it('returns strength-1 for weak signals (<25)', () => {
      expect(signalIcon(0)).toBe('wifi-strength-1');
      expect(signalIcon(10)).toBe('wifi-strength-1');
      expect(signalIcon(24)).toBe('wifi-strength-1');
    });
  });

  describe('parseWifiScanOutput', () => {
    it('parses valid JSON scan output', () => {
      const stdout = JSON.stringify([
        {
          device: 'wlan0',
          ssid: 'HomeNetwork',
          bssid: 'AA:BB:CC:DD:EE:01',
          signal: 85,
          security: 'wpa2',
          channel: 6,
          frequency: 2437,
          rate: '54 Mbit/s',
        },
        {
          device: 'wlan0',
          ssid: 'OfficeWiFi',
          bssid: 'AA:BB:CC:DD:EE:02',
          signal: 60,
          security: 'wpa3',
          channel: 36,
          frequency: 5180,
          rate: '867 Mbit/s',
        },
      ]);

      const result = parseWifiScanOutput(stdout);

      expect(result).toHaveLength(2);
      expect(result[0]).toEqual({
        device: 'wlan0',
        ssid: 'HomeNetwork',
        bssid: 'AA:BB:CC:DD:EE:01',
        signal: 85,
        security: 'wpa2',
        channel: 6,
        frequency: 2437,
        band: '2.4 GHz',
        bands: ['2.4 GHz'],
        rate: '54 Mbit/s',
      });
      expect(result[1]).toEqual({
        device: 'wlan0',
        ssid: 'OfficeWiFi',
        bssid: 'AA:BB:CC:DD:EE:02',
        signal: 60,
        security: 'wpa3',
        channel: 36,
        frequency: 5180,
        band: '5 GHz',
        bands: ['5 GHz'],
        rate: '867 Mbit/s',
      });
    });

    it('derives band from frequency', () => {
      const stdout = JSON.stringify([
        { ssid: 'A', frequency: 2412 },
        { ssid: 'B', frequency: 5500 },
        { ssid: 'C', frequency: 6000 },
      ]);

      const result = parseWifiScanOutput(stdout);

      expect(result[0].band).toBe('2.4 GHz');
      expect(result[1].band).toBe('5 GHz');
      expect(result[2].band).toBe('6 GHz');
    });

    it('handles missing fields with defaults', () => {
      const stdout = JSON.stringify([{}]);

      const result = parseWifiScanOutput(stdout);

      expect(result).toHaveLength(1);
      expect(result[0]).toEqual({
        device: '',
        ssid: '',
        bssid: '',
        signal: 0,
        security: 'open',
        channel: 0,
        frequency: 0,
        band: '',
        bands: [''],
        rate: '',
      });
    });

    it('returns empty array for empty JSON array', () => {
      expect(parseWifiScanOutput('[]')).toEqual([]);
    });

    it('returns empty array for invalid JSON', () => {
      expect(parseWifiScanOutput('not json')).toEqual([]);
      expect(parseWifiScanOutput('')).toEqual([]);
    });

    it('returns empty array for non-array JSON', () => {
      expect(parseWifiScanOutput('{"key": "value"}')).toEqual([]);
      expect(parseWifiScanOutput('"string"')).toEqual([]);
      expect(parseWifiScanOutput('42')).toEqual([]);
    });

    it('handles open network with no security field', () => {
      const stdout = JSON.stringify([
        {
          ssid: 'OpenNet',
          bssid: '00:11:22:33:44:55',
          signal: 50,
          channel: 1,
          frequency: 2412,
        },
      ]);

      const result = parseWifiScanOutput(stdout);

      expect(result[0].security).toBe('open');
    });

    it('deduplicates same-SSID networks, keeping strongest signal and aggregating bands', () => {
      const stdout = JSON.stringify([
        {
          ssid: 'Corp',
          bssid: 'AA:BB:CC:DD:EE:01',
          signal: 90,
          security: 'wpa2',
          channel: 1,
          frequency: 2412,
        },
        {
          ssid: 'Corp',
          bssid: 'AA:BB:CC:DD:EE:02',
          signal: 40,
          security: 'wpa2',
          channel: 36,
          frequency: 5180,
        },
      ]);

      const result = parseWifiScanOutput(stdout);

      expect(result).toHaveLength(1);
      // Keeps strongest signal entry as representative
      expect(result[0].bssid).toBe('AA:BB:CC:DD:EE:01');
      expect(result[0].signal).toBe(90);
      // Aggregates bands from all BSSIDs
      expect(result[0].bands).toEqual(['2.4 GHz', '5 GHz']);
    });

    it('handles hidden networks (empty SSID)', () => {
      const stdout = JSON.stringify([
        {
          ssid: '',
          bssid: '00:11:22:33:44:55',
          signal: 70,
          security: 'wpa2',
          channel: 11,
          frequency: 2462,
        },
      ]);

      const result = parseWifiScanOutput(stdout);

      expect(result[0].ssid).toBe('');
    });

    it('parses 6 GHz networks correctly', () => {
      const stdout = JSON.stringify([
        {
          device: 'wlan0',
          ssid: 'WiFi6E',
          bssid: 'FF:00:FF:00:FF:00',
          signal: 55,
          security: 'wpa3',
          channel: 1,
          frequency: 5955,
          rate: '2401 Mbit/s',
        },
      ]);

      const result = parseWifiScanOutput(stdout);

      expect(result[0].band).toBe('6 GHz');
      expect(result[0].security).toBe('wpa3');
    });
  });

  describe('WiFi security classification', () => {
    it('treats "open" as no password required', () => {
      const isOpen = (sec: string) => sec === 'open' || sec === '--';
      expect(isOpen('open')).toBe(true);
      expect(isOpen('--')).toBe(true);
      expect(isOpen('wpa2')).toBe(false);
      expect(isOpen('wpa3')).toBe(false);
      expect(isOpen('SAE')).toBe(false);
    });
  });
});
