import React from 'react';
import { DeviceProvider } from './DeviceContext';
import { AuthProvider } from './AuthContext';
import { CertificateProvider } from './CertificateContext';

interface AppProviderProps {
  children: React.ReactNode;
}

/**
 * AppProvider
 *
 * Wraps all context providers in the correct order to manage global state.
 * Order matters: Certificate → Auth → Device (inner providers may depend on outer ones)
 */
export function AppProvider({ children }: AppProviderProps) {
  return (
    <CertificateProvider>
      <AuthProvider>
        <DeviceProvider>{children}</DeviceProvider>
      </AuthProvider>
    </CertificateProvider>
  );
}
