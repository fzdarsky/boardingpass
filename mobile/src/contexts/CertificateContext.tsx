import React, { createContext, useContext, useReducer, useCallback } from 'react';

// Will be imported from types/certificate.ts once created
interface CertificateInfo {
  deviceId: string;
  fingerprint: string;
  subject: string;
  issuer: string;
  validFrom: Date;
  validTo: Date;
  isSelfSigned: boolean;
  isPinned: boolean;
  trustStatus: 'trusted' | 'new' | 'changed' | 'untrusted';
  pinnedAt?: Date;
}

interface CertificateState {
  certificates: Map<string, CertificateInfo>;
  pendingTrustDecisions: Map<string, CertificateInfo>;
}

type CertificateAction =
  | { type: 'ADD_CERTIFICATE'; payload: CertificateInfo }
  | { type: 'PIN_CERTIFICATE'; payload: { deviceId: string; pinnedAt: Date } }
  | { type: 'REMOVE_CERTIFICATE'; payload: string }
  | { type: 'ADD_PENDING_TRUST'; payload: CertificateInfo }
  | { type: 'REMOVE_PENDING_TRUST'; payload: string }
  | { type: 'CLEAR_ALL_CERTIFICATES' };

interface CertificateContextValue {
  state: CertificateState;
  addCertificate: (cert: CertificateInfo) => void;
  pinCertificate: (deviceId: string) => void;
  removeCertificate: (deviceId: string) => void;
  addPendingTrust: (cert: CertificateInfo) => void;
  removePendingTrust: (deviceId: string) => void;
  clearAllCertificates: () => void;
  getCertificateByDeviceId: (deviceId: string) => CertificateInfo | undefined;
  isPinned: (deviceId: string) => boolean;
}

const CertificateContext = createContext<CertificateContextValue | undefined>(undefined);

const initialState: CertificateState = {
  certificates: new Map(),
  pendingTrustDecisions: new Map(),
};

function certificateReducer(state: CertificateState, action: CertificateAction): CertificateState {
  switch (action.type) {
    case 'ADD_CERTIFICATE': {
      const newCerts = new Map(state.certificates);
      newCerts.set(action.payload.deviceId, action.payload);
      return { ...state, certificates: newCerts };
    }

    case 'PIN_CERTIFICATE': {
      const cert = state.certificates.get(action.payload.deviceId);
      if (!cert) return state;

      const newCerts = new Map(state.certificates);
      newCerts.set(action.payload.deviceId, {
        ...cert,
        isPinned: true,
        trustStatus: 'trusted',
        pinnedAt: action.payload.pinnedAt,
      });
      return { ...state, certificates: newCerts };
    }

    case 'REMOVE_CERTIFICATE': {
      const newCerts = new Map(state.certificates);
      newCerts.delete(action.payload);
      return { ...state, certificates: newCerts };
    }

    case 'ADD_PENDING_TRUST': {
      const newPending = new Map(state.pendingTrustDecisions);
      newPending.set(action.payload.deviceId, action.payload);
      return { ...state, pendingTrustDecisions: newPending };
    }

    case 'REMOVE_PENDING_TRUST': {
      const newPending = new Map(state.pendingTrustDecisions);
      newPending.delete(action.payload);
      return { ...state, pendingTrustDecisions: newPending };
    }

    case 'CLEAR_ALL_CERTIFICATES':
      return { ...state, certificates: new Map(), pendingTrustDecisions: new Map() };

    default:
      return state;
  }
}

export function CertificateProvider({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(certificateReducer, initialState);

  const addCertificate = useCallback((cert: CertificateInfo) => {
    dispatch({ type: 'ADD_CERTIFICATE', payload: cert });
  }, []);

  const pinCertificate = useCallback((deviceId: string) => {
    dispatch({ type: 'PIN_CERTIFICATE', payload: { deviceId, pinnedAt: new Date() } });
  }, []);

  const removeCertificate = useCallback((deviceId: string) => {
    dispatch({ type: 'REMOVE_CERTIFICATE', payload: deviceId });
  }, []);

  const addPendingTrust = useCallback((cert: CertificateInfo) => {
    dispatch({ type: 'ADD_PENDING_TRUST', payload: cert });
  }, []);

  const removePendingTrust = useCallback((deviceId: string) => {
    dispatch({ type: 'REMOVE_PENDING_TRUST', payload: deviceId });
  }, []);

  const clearAllCertificates = useCallback(() => {
    dispatch({ type: 'CLEAR_ALL_CERTIFICATES' });
  }, []);

  const getCertificateByDeviceId = useCallback(
    (deviceId: string) => {
      return state.certificates.get(deviceId);
    },
    [state.certificates]
  );

  const isPinned = useCallback(
    (deviceId: string) => {
      const cert = state.certificates.get(deviceId);
      return cert?.isPinned ?? false;
    },
    [state.certificates]
  );

  const value: CertificateContextValue = {
    state,
    addCertificate,
    pinCertificate,
    removeCertificate,
    addPendingTrust,
    removePendingTrust,
    clearAllCertificates,
    getCertificateByDeviceId,
    isPinned,
  };

  return <CertificateContext.Provider value={value}>{children}</CertificateContext.Provider>;
}

export function useCertificate() {
  const context = useContext(CertificateContext);
  if (context === undefined) {
    throw new Error('useCertificate must be used within a CertificateProvider');
  }
  return context;
}
