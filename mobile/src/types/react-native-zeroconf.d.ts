/**
 * Type declarations for react-native-zeroconf
 *
 * react-native-zeroconf doesn't provide official TypeScript definitions.
 * This file provides minimal type coverage for the API we use.
 */

declare module 'react-native-zeroconf' {
  export interface Service {
    name: string;
    host: string;
    port: number;
    addresses: string[];
    txt?: Record<string, string>;
  }

  export default class Zeroconf {
    constructor();

    scan(type: string, protocol: string, domain: string): void;
    stop(): void;

    on(event: 'found', listener: (service: Service) => void): void;
    on(event: 'resolved', listener: (service: Service) => void): void;
    on(event: 'remove', listener: (service: Service) => void): void;
    on(event: 'error', listener: (error: unknown) => void): void;
    on(event: 'start' | 'stop' | 'update', listener: () => void): void;

    removeListener(event: string, listener: (...args: unknown[]) => void): void;
  }
}
