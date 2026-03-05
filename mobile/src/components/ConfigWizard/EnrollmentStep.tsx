/**
 * EnrollmentStep Component (Step 5)
 *
 * Optional Insights enrollment (endpoint, Org ID, Activation Key).
 * Optional Flight Control enrollment (endpoint, username, password).
 */

import React, { useCallback } from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, TextInput, Switch, Divider, HelperText, useTheme } from 'react-native-paper';
import { useWizard } from '../../contexts/WizardContext';
import { validateHttpsUrl } from '../../utils/network-validation';
import type { InsightsConfig, FlightControlConfig } from '../../types/wizard';
import { DEFAULT_INSIGHTS_ENDPOINT } from '../../types/wizard';
import { spacing } from '../../theme';

export default function EnrollmentStep() {
  const { state, updateEnrollment } = useWizard();
  const theme = useTheme();

  const insights = state.enrollment.insights;
  const flightControl = state.enrollment.flightControl;

  const toggleInsights = useCallback(
    (enabled: boolean) => {
      updateEnrollment({
        ...state.enrollment,
        insights: enabled
          ? { endpoint: DEFAULT_INSIGHTS_ENDPOINT, orgId: '', activationKey: '' }
          : null,
      });
    },
    [state.enrollment, updateEnrollment]
  );

  const updateInsights = useCallback(
    (updates: Partial<InsightsConfig>) => {
      if (!insights) return;
      updateEnrollment({
        ...state.enrollment,
        insights: { ...insights, ...updates },
      });
    },
    [insights, state.enrollment, updateEnrollment]
  );

  const toggleFlightControl = useCallback(
    (enabled: boolean) => {
      updateEnrollment({
        ...state.enrollment,
        flightControl: enabled ? { endpoint: '', username: '', password: '' } : null,
      });
    },
    [state.enrollment, updateEnrollment]
  );

  const updateFlightControl = useCallback(
    (updates: Partial<FlightControlConfig>) => {
      if (!flightControl) return;
      updateEnrollment({
        ...state.enrollment,
        flightControl: { ...flightControl, ...updates },
      });
    },
    [flightControl, state.enrollment, updateEnrollment]
  );

  return (
    <View style={styles.container}>
      <Text variant="titleMedium" style={[styles.title, { color: theme.colors.onSurface }]}>
        Enrollment Server Registration
      </Text>

      <Text
        variant="bodyMedium"
        style={[styles.description, { color: theme.colors.onSurfaceVariant }]}
      >
        Optionally register this device with management services. Both can be skipped.
      </Text>

      {/* Insights Section */}
      <View style={styles.sectionHeader}>
        <Text variant="titleSmall" style={{ color: theme.colors.primary }}>
          Red Hat Insights
        </Text>
        <Switch
          value={insights !== null}
          onValueChange={toggleInsights}
          accessibilityLabel="Enable Red Hat Insights enrollment"
        />
      </View>

      {insights && (
        <View style={styles.fields}>
          <TextInput
            label="Endpoint URL"
            value={insights.endpoint}
            onChangeText={v => updateInsights({ endpoint: v })}
            mode="outlined"
            autoCapitalize="none"
            autoCorrect={false}
            placeholder={DEFAULT_INSIGHTS_ENDPOINT}
            accessibilityLabel="Insights endpoint URL"
            style={styles.input}
            error={!!validateHttpsUrl(insights.endpoint)}
          />
          {validateHttpsUrl(insights.endpoint) && (
            <HelperText type="error" visible={true}>
              {validateHttpsUrl(insights.endpoint)}
            </HelperText>
          )}

          <TextInput
            label="Organisation ID"
            value={insights.orgId}
            onChangeText={v => updateInsights({ orgId: v })}
            mode="outlined"
            autoCapitalize="none"
            accessibilityLabel="Organisation ID"
            style={styles.input}
          />
          {!insights.orgId && (
            <HelperText type="info" visible={true}>
              Required for Insights enrollment
            </HelperText>
          )}

          <TextInput
            label="Activation Key"
            value={insights.activationKey}
            onChangeText={v => updateInsights({ activationKey: v })}
            mode="outlined"
            secureTextEntry
            autoCapitalize="none"
            accessibilityLabel="Activation key"
            style={styles.input}
          />
          {!insights.activationKey && (
            <HelperText type="info" visible={true}>
              Required for Insights enrollment
            </HelperText>
          )}
        </View>
      )}

      {insights && flightControl && (
        <HelperText type="info" visible={true} style={styles.managementNote}>
          Remote management will be handled by Flight Control. Insights will be used for analytics
          only.
        </HelperText>
      )}

      <Divider style={styles.divider} />

      {/* Flight Control Section */}
      <View style={styles.sectionHeader}>
        <Text variant="titleSmall" style={{ color: theme.colors.primary }}>
          Flight Control
        </Text>
        <Switch
          value={flightControl !== null}
          onValueChange={toggleFlightControl}
          accessibilityLabel="Enable Flight Control enrollment"
        />
      </View>

      {flightControl && (
        <View style={styles.fields}>
          <TextInput
            label="Endpoint URL"
            value={flightControl.endpoint}
            onChangeText={v => updateFlightControl({ endpoint: v })}
            mode="outlined"
            autoCapitalize="none"
            autoCorrect={false}
            placeholder="https://flightctl.example.com"
            accessibilityLabel="Flight Control endpoint URL"
            style={styles.input}
            error={!!flightControl.endpoint && !!validateHttpsUrl(flightControl.endpoint)}
          />
          {flightControl.endpoint && validateHttpsUrl(flightControl.endpoint) && (
            <HelperText type="error" visible={true}>
              {validateHttpsUrl(flightControl.endpoint)}
            </HelperText>
          )}

          <TextInput
            label="Username"
            value={flightControl.username}
            onChangeText={v => updateFlightControl({ username: v })}
            mode="outlined"
            autoCapitalize="none"
            accessibilityLabel="Flight Control username"
            style={styles.input}
          />

          <TextInput
            label="Password"
            value={flightControl.password}
            onChangeText={v => updateFlightControl({ password: v })}
            mode="outlined"
            secureTextEntry
            accessibilityLabel="Flight Control password"
            style={styles.input}
          />
        </View>
      )}
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
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: spacing.sm,
  },
  fields: {
    paddingLeft: spacing.md,
    marginBottom: spacing.sm,
  },
  input: {
    marginBottom: spacing.xs,
  },
  managementNote: {
    paddingLeft: spacing.md,
  },
  divider: {
    marginVertical: spacing.md,
  },
});
