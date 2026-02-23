/**
 * ErrorBoundary Component
 *
 * React Error Boundary to catch and handle rendering errors gracefully.
 * Enhanced with recovery actions: retry, navigate back, and custom error views.
 * Implements FR-024 (error messages), FR-025 (retry mechanism)
 */

import React, { Component, ReactNode } from 'react';
import { View, Text, StyleSheet, ScrollView } from 'react-native';
import { Button, Icon } from 'react-native-paper';
import { router } from 'expo-router';
import { toAppError, getErrorMessage, getErrorTitle } from '../../utils/error-messages';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: (error: Error, resetError: () => void, goBack: () => void) => ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  enableNavigation?: boolean;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorCount: number;
}

/**
 * ErrorBoundary
 *
 * Catches React rendering errors and displays fallback UI.
 * Provides reset functionality to recover from errors.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorCount: 0,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    // Log error to console (could integrate with error tracking service)
    console.error('ErrorBoundary caught error:', error, errorInfo);

    // Track error count
    this.setState((prev) => ({
      errorCount: prev.errorCount + 1,
    }));

    // Call custom error handler if provided
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }

    // Could send to error tracking service here
    // e.g., Sentry.captureException(error, { extra: errorInfo });
  }

  resetError = (): void => {
    this.setState({
      hasError: false,
      error: null,
    });
  };

  goBack = (): void => {
    this.resetError();
    if (this.props.enableNavigation !== false && router.canGoBack()) {
      router.back();
    }
  };

  render(): ReactNode {
    if (this.state.hasError && this.state.error) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback(this.state.error, this.resetError, this.goBack);
      }

      const appError = toAppError(this.state.error);
      const errorTitle = getErrorTitle(appError);
      const errorMessage = getErrorMessage(appError);
      const canGoBack = this.props.enableNavigation !== false && router.canGoBack();

      // Default fallback UI with enhanced recovery actions
      return (
        <ScrollView contentContainerStyle={styles.scrollContainer}>
          <View style={styles.container}>
            <View style={styles.iconContainer}>
              <Icon source="alert-circle" size={64} color="#D32F2F" />
            </View>

            <View style={styles.content}>
              <Text style={styles.title}>{errorTitle}</Text>
              <Text style={styles.message}>{errorMessage}</Text>

              {/* Multiple error warning */}
              {this.state.errorCount > 1 && (
                <View style={styles.warningBox}>
                  <Icon source="alert" size={20} color="#FF9800" />
                  <Text style={styles.warningText}>
                    This error has occurred {this.state.errorCount} times. If it persists, try
                    going back or restarting the app.
                  </Text>
                </View>
              )}

              {/* Development error details */}
              {__DEV__ && (
                <View style={styles.errorDetails}>
                  <Text style={styles.errorTitle}>Error Details (Development Only):</Text>
                  <Text style={styles.errorText}>{this.state.error.message}</Text>
                  {this.state.error.stack && (
                    <ScrollView style={styles.stackContainer} nestedScrollEnabled>
                      <Text style={styles.stackTrace}>{this.state.error.stack}</Text>
                    </ScrollView>
                  )}
                </View>
              )}

              {/* Recovery actions */}
              <View style={styles.actions}>
                <Button
                  mode="contained"
                  onPress={this.resetError}
                  style={styles.button}
                  icon="refresh"
                >
                  Try Again
                </Button>

                {canGoBack && (
                  <Button
                    mode="outlined"
                    onPress={this.goBack}
                    style={styles.button}
                    icon="arrow-left"
                  >
                    Go Back
                  </Button>
                )}
              </View>

              <Text style={styles.footerText}>
                If this problem persists, please restart the app
              </Text>
            </View>
          </View>
        </ScrollView>
      );
    }

    return this.props.children;
  }
}

const styles = StyleSheet.create({
  scrollContainer: {
    flexGrow: 1,
    justifyContent: 'center',
  },
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#fff',
    padding: 24,
  },
  iconContainer: {
    marginBottom: 24,
  },
  content: {
    maxWidth: 400,
    width: '100%',
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#333',
    marginBottom: 12,
    textAlign: 'center',
  },
  message: {
    fontSize: 16,
    color: '#666',
    marginBottom: 24,
    textAlign: 'center',
    lineHeight: 24,
  },
  warningBox: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    backgroundColor: '#FFF3E0',
    padding: 16,
    borderRadius: 8,
    marginBottom: 16,
  },
  warningText: {
    flex: 1,
    marginLeft: 12,
    color: '#E65100',
    fontSize: 14,
    lineHeight: 20,
  },
  errorDetails: {
    backgroundColor: '#f5f5f5',
    padding: 16,
    borderRadius: 8,
    marginBottom: 24,
  },
  errorTitle: {
    fontSize: 14,
    fontWeight: 'bold',
    color: '#d32f2f',
    marginBottom: 8,
  },
  errorText: {
    fontSize: 12,
    color: '#666',
    fontFamily: 'monospace',
    marginBottom: 8,
  },
  stackContainer: {
    maxHeight: 150,
  },
  stackTrace: {
    fontSize: 10,
    color: '#999',
    fontFamily: 'monospace',
  },
  actions: {
    width: '100%',
    gap: 12,
  },
  button: {
    marginBottom: 8,
  },
  footerText: {
    marginTop: 16,
    textAlign: 'center',
    color: '#999',
    fontSize: 14,
    lineHeight: 20,
  },
});

export default ErrorBoundary;
