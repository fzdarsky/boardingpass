import React, { createContext, useContext, useReducer, useCallback } from 'react';

// Will be imported from types/auth.ts once created
interface AuthenticationSession {
  deviceId: string;
  token: string;
  expiresAt: Date;
  createdAt: Date;
}

interface AuthState {
  sessions: Map<string, AuthenticationSession>;
  activeDeviceId: string | null;
  isAuthenticating: boolean;
  authError: string | null;
}

type AuthAction =
  | { type: 'SET_SESSION'; payload: AuthenticationSession }
  | { type: 'REMOVE_SESSION'; payload: string }
  | { type: 'SET_ACTIVE_DEVICE'; payload: string | null }
  | { type: 'SET_AUTHENTICATING'; payload: boolean }
  | { type: 'SET_AUTH_ERROR'; payload: string | null }
  | { type: 'CLEAR_ALL_SESSIONS' };

interface AuthContextValue {
  state: AuthState;
  setSession: (session: AuthenticationSession) => void;
  removeSession: (deviceId: string) => void;
  setActiveDevice: (deviceId: string | null) => void;
  setAuthenticating: (authenticating: boolean) => void;
  setAuthError: (error: string | null) => void;
  clearAllSessions: () => void;
  getSessionByDeviceId: (deviceId: string) => AuthenticationSession | undefined;
  isSessionValid: (deviceId: string) => boolean;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

const initialState: AuthState = {
  sessions: new Map(),
  activeDeviceId: null,
  isAuthenticating: false,
  authError: null,
};

function authReducer(state: AuthState, action: AuthAction): AuthState {
  switch (action.type) {
    case 'SET_SESSION': {
      const newSessions = new Map(state.sessions);
      newSessions.set(action.payload.deviceId, action.payload);
      return { ...state, sessions: newSessions, authError: null };
    }

    case 'REMOVE_SESSION': {
      const newSessions = new Map(state.sessions);
      newSessions.delete(action.payload);
      const activeDeviceId = state.activeDeviceId === action.payload ? null : state.activeDeviceId;
      return { ...state, sessions: newSessions, activeDeviceId };
    }

    case 'SET_ACTIVE_DEVICE':
      return { ...state, activeDeviceId: action.payload };

    case 'SET_AUTHENTICATING':
      return { ...state, isAuthenticating: action.payload };

    case 'SET_AUTH_ERROR':
      return { ...state, authError: action.payload };

    case 'CLEAR_ALL_SESSIONS':
      return { ...state, sessions: new Map(), activeDeviceId: null, authError: null };

    default:
      return state;
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(authReducer, initialState);

  const setSession = useCallback((session: AuthenticationSession) => {
    dispatch({ type: 'SET_SESSION', payload: session });
  }, []);

  const removeSession = useCallback((deviceId: string) => {
    dispatch({ type: 'REMOVE_SESSION', payload: deviceId });
  }, []);

  const setActiveDevice = useCallback((deviceId: string | null) => {
    dispatch({ type: 'SET_ACTIVE_DEVICE', payload: deviceId });
  }, []);

  const setAuthenticating = useCallback((authenticating: boolean) => {
    dispatch({ type: 'SET_AUTHENTICATING', payload: authenticating });
  }, []);

  const setAuthError = useCallback((error: string | null) => {
    dispatch({ type: 'SET_AUTH_ERROR', payload: error });
  }, []);

  const clearAllSessions = useCallback(() => {
    dispatch({ type: 'CLEAR_ALL_SESSIONS' });
  }, []);

  const getSessionByDeviceId = useCallback(
    (deviceId: string) => {
      return state.sessions.get(deviceId);
    },
    [state.sessions]
  );

  const isSessionValid = useCallback(
    (deviceId: string) => {
      const session = state.sessions.get(deviceId);
      if (!session) return false;
      return new Date() < new Date(session.expiresAt);
    },
    [state.sessions]
  );

  const value: AuthContextValue = {
    state,
    setSession,
    removeSession,
    setActiveDevice,
    setAuthenticating,
    setAuthError,
    clearAllSessions,
    getSessionByDeviceId,
    isSessionValid,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
