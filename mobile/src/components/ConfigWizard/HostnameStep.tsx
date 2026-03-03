/**
 * HostnameStep Component (Step 1)
 *
 * TextInput pre-populated from device info with RFC 1123 validation
 * and inline error display.
 */

import React, { useCallback, useState } from 'react';
import { View, StyleSheet } from 'react-native';
import { TextInput, Text, HelperText, useTheme } from 'react-native-paper';
import { useWizard } from '../../contexts/WizardContext';
import { validateHostname } from '../../utils/network-validation';
import { spacing } from '../../theme';

export default function HostnameStep() {
  const { state, updateHostname } = useWizard();
  const theme = useTheme();
  const [touched, setTouched] = useState(false);

  const error = touched ? validateHostname(state.hostname.hostname) : null;

  const handleChange = useCallback(
    (text: string) => {
      updateHostname({ hostname: text.toLowerCase().trim() });
    },
    [updateHostname]
  );

  const handleBlur = useCallback(() => {
    setTouched(true);
  }, []);

  return (
    <View style={styles.container}>
      <Text variant="titleMedium" style={[styles.title, { color: theme.colors.onSurface }]}>
        Set Device Hostname
      </Text>

      <Text
        variant="bodyMedium"
        style={[styles.description, { color: theme.colors.onSurfaceVariant }]}
      >
        Enter the hostname for this device. It must follow RFC 1123 naming rules: alphanumeric
        characters and hyphens, 1-63 characters per label.
      </Text>

      <TextInput
        label="Hostname"
        value={state.hostname.hostname}
        onChangeText={handleChange}
        onBlur={handleBlur}
        mode="outlined"
        error={!!error}
        autoCapitalize="none"
        autoCorrect={false}
        placeholder="e.g., edge-device-001"
        accessibilityLabel="Device hostname"
        style={styles.input}
      />

      <HelperText type="error" visible={!!error}>
        {error}
      </HelperText>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    paddingHorizontal: spacing.md,
  },
  title: {
    marginBottom: spacing.sm,
  },
  description: {
    marginBottom: spacing.lg,
  },
  input: {
    marginBottom: spacing.xs,
  },
});
