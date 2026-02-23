/**
 * Error Context
 * Provides global error handling with toast notifications and dialogs
 * Implements FR-108 (global error toasts), FR-109 (error dialogs)
 */

import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { Snackbar, Portal, Dialog, Button, Text } from 'react-native-paper';
import { AppError, getErrorMessage, getErrorTitle, getErrorAction } from '../utils/error-messages';

interface ErrorContextValue {
  /**
   * Show a non-critical error as a toast notification
   */
  showToast: (error: AppError | string, duration?: number) => void;

  /**
   * Show a critical error as a blocking dialog
   */
  showDialog: (error: AppError | string, onDismiss?: () => void, onRetry?: () => void) => void;

  /**
   * Dismiss current toast
   */
  dismissToast: () => void;

  /**
   * Dismiss current dialog
   */
  dismissDialog: () => void;
}

const ErrorContext = createContext<ErrorContextValue | undefined>(undefined);

interface ErrorProviderProps {
  children: ReactNode;
}

export const ErrorProvider: React.FC<ErrorProviderProps> = ({ children }) => {
  const [toastVisible, setToastVisible] = useState(false);
  const [toastMessage, setToastMessage] = useState('');
  const [toastDuration, setToastDuration] = useState(4000);

  const [dialogVisible, setDialogVisible] = useState(false);
  const [dialogTitle, setDialogTitle] = useState('');
  const [dialogMessage, setDialogMessage] = useState('');
  const [dialogAction, setDialogAction] = useState('Dismiss');
  const [dialogOnDismiss, setDialogOnDismiss] = useState<(() => void) | undefined>();
  const [dialogOnRetry, setDialogOnRetry] = useState<(() => void) | undefined>();

  const showToast = useCallback((error: AppError | string, duration: number = 4000) => {
    const message = typeof error === 'string' ? error : getErrorMessage(error);
    setToastMessage(message);
    setToastDuration(duration);
    setToastVisible(true);
  }, []);

  const showDialog = useCallback(
    (error: AppError | string, onDismiss?: () => void, onRetry?: () => void) => {
      if (typeof error === 'string') {
        setDialogTitle('Error');
        setDialogMessage(error);
        setDialogAction('Dismiss');
      } else {
        setDialogTitle(getErrorTitle(error));
        setDialogMessage(getErrorMessage(error));
        setDialogAction(getErrorAction(error));
      }

      setDialogOnDismiss(() => onDismiss);
      setDialogOnRetry(() => onRetry);
      setDialogVisible(true);
    },
    []
  );

  const dismissToast = useCallback(() => {
    setToastVisible(false);
  }, []);

  const dismissDialog = useCallback(() => {
    setDialogVisible(false);
    if (dialogOnDismiss) {
      dialogOnDismiss();
    }
  }, [dialogOnDismiss]);

  const handleRetry = useCallback(() => {
    setDialogVisible(false);
    if (dialogOnRetry) {
      dialogOnRetry();
    }
  }, [dialogOnRetry]);

  return (
    <ErrorContext.Provider value={{ showToast, showDialog, dismissToast, dismissDialog }}>
      {children}

      <Portal>
        {/* Toast for non-critical errors */}
        <Snackbar
          visible={toastVisible}
          onDismiss={dismissToast}
          duration={toastDuration}
          action={{
            label: 'Dismiss',
            onPress: dismissToast,
          }}
          style={{ backgroundColor: '#D32F2F' }}
        >
          {toastMessage}
        </Snackbar>

        {/* Dialog for critical errors */}
        <Dialog visible={dialogVisible} onDismiss={dismissDialog}>
          <Dialog.Title>{dialogTitle}</Dialog.Title>
          <Dialog.Content>
            <Text>{dialogMessage}</Text>
          </Dialog.Content>
          <Dialog.Actions>
            <Button onPress={dismissDialog}>Cancel</Button>
            {dialogOnRetry && <Button onPress={handleRetry}>{dialogAction}</Button>}
          </Dialog.Actions>
        </Dialog>
      </Portal>
    </ErrorContext.Provider>
  );
};

export const useErrorContext = (): ErrorContextValue => {
  const context = useContext(ErrorContext);
  if (!context) {
    throw new Error('useErrorContext must be used within ErrorProvider');
  }
  return context;
};

export default ErrorProvider;
