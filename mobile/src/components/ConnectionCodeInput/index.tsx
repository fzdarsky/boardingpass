/**
 * ConnectionCodeInput Component
 *
 * Manual input field for connection codes with validation and formatting.
 * Handles input validation, error display, and paste functionality.
 *
 * Features:
 * - Real-time validation (FR-027)
 * - Clear error messages (FR-024)
 * - Paste support for easy code entry
 * - Accessible UI with proper labels
 *
 * Usage:
 * ```tsx
 * <ConnectionCodeInput
 *   value={code}
 *   onChangeText={setCode}
 *   onSubmit={handleAuthenticate}
 *   error={error}
 *   disabled={isAuthenticating}
 * />
 * ```
 */

import React, { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { TextInput, HelperText } from 'react-native-paper';
import { validateAndSanitizeConnectionCode } from '../../utils/validation';

/**
 * ConnectionCodeInput Props
 */
export interface ConnectionCodeInputProps {
  /**
   * Current connection code value
   */
  value: string;

  /**
   * Callback when code changes
   */
  onChangeText: (code: string) => void;

  /**
   * Callback when submit button is pressed
   */
  onSubmit?: () => void;

  /**
   * External error message (e.g., from authentication failure)
   */
  error?: string;

  /**
   * Whether input is disabled (e.g., during authentication)
   */
  disabled?: boolean;

  /**
   * Show validation in real-time (default: false, only on blur)
   */
  liveValidation?: boolean;
}

/**
 * ConnectionCodeInput Component
 */
export default function ConnectionCodeInput({
  value,
  onChangeText,
  onSubmit,
  error: externalError,
  disabled = false,
  liveValidation = false,
}: ConnectionCodeInputProps): React.ReactElement {
  const [isFocused, setIsFocused] = useState(false);
  const [hasBlurred, setHasBlurred] = useState(false);
  const [validationError, setValidationError] = useState<string | undefined>();

  /**
   * Handle text change with validation
   */
  const handleChangeText = (text: string) => {
    onChangeText(text);

    // Clear external error when user starts typing
    if (externalError && text !== value) {
      // External error will be cleared by parent
    }

    // Live validation (if enabled)
    if (liveValidation || hasBlurred) {
      validateInput(text);
    }
  };

  /**
   * Validate input
   */
  const validateInput = (code: string) => {
    if (code.trim().length === 0) {
      setValidationError(undefined);
      return;
    }

    const result = validateAndSanitizeConnectionCode(code);
    setValidationError(result.valid ? undefined : result.error);
  };

  /**
   * Handle focus
   */
  const handleFocus = () => {
    setIsFocused(true);
  };

  /**
   * Handle blur (validate on blur)
   */
  const handleBlur = () => {
    setIsFocused(false);
    setHasBlurred(true);
    validateInput(value);
  };

  /**
   * Handle submit (Enter key or submit button)
   */
  const handleSubmit = () => {
    // Validate before submit
    validateInput(value);

    const result = validateAndSanitizeConnectionCode(value);
    if (result.valid && onSubmit) {
      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[ConnectionCodeInput] Valid code submitted', {
          codeLength: result.sanitized.length,
          note: 'Code value NEVER logged (FR-029)',
        });
      }
      onSubmit();
    } else if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[ConnectionCodeInput] Invalid code - submit blocked', {
        error: result.error,
      });
    }
  };

  /**
   * Clear input
   */
  const handleClear = () => {
    onChangeText('');
    setValidationError(undefined);
  };

  /**
   * Determine which error to display
   */
  const displayError = externalError || validationError;
  const hasError = Boolean(displayError);

  return (
    <View style={styles.container}>
      <TextInput
        label="Connection Code"
        value={value}
        onChangeText={handleChangeText}
        onFocus={handleFocus}
        onBlur={handleBlur}
        onSubmitEditing={handleSubmit}
        mode="outlined"
        error={hasError}
        disabled={disabled}
        autoCapitalize="none"
        autoCorrect={false}
        autoComplete="off"
        textContentType="none"
        secureTextEntry={false}
        selectTextOnFocus={true}
        placeholder="Enter or paste connection code"
        accessibilityLabel="Connection code input"
        accessibilityHint="Enter the connection code from the device"
        returnKeyType="done"
        blurOnSubmit={true}
        style={styles.input}
        right={
          value.length > 0 && !disabled ? (
            <TextInput.Icon icon="close-circle" onPress={handleClear} />
          ) : undefined
        }
      />

      {/* Helper text / Error message */}
      <HelperText type={hasError ? 'error' : 'info'} visible={Boolean(displayError || isFocused)}>
        {displayError || 'Enter the connection code displayed on the device'}
      </HelperText>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    width: '100%',
  },
  input: {
    fontSize: 16,
    fontFamily: 'monospace', // Use monospace font for codes
  },
});
