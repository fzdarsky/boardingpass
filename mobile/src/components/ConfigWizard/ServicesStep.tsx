/**
 * ServicesStep Component (Step 4)
 *
 * NTP switch (Automatic/Manual) with manual server list input.
 * Optional proxy section (hostname, port, optional username/password).
 */

import React, { useCallback, useState } from 'react';
import { View, StyleSheet } from 'react-native';
import {
  Text,
  TextInput,
  Switch,
  Divider,
  IconButton,
  HelperText,
  useTheme,
} from 'react-native-paper';
import { useWizard } from '../../contexts/WizardContext';
import { validateNtpServer, validatePort } from '../../utils/network-validation';
import type { NTPConfig, ProxyConfig } from '../../types/wizard';
import { spacing } from '../../theme';

export default function ServicesStep() {
  const { state, updateServices } = useWizard();
  const theme = useTheme();
  const [newServer, setNewServer] = useState('');

  const updateNtp = useCallback(
    (updates: Partial<NTPConfig>) => {
      updateServices({
        ...state.services,
        ntp: { ...state.services.ntp, ...updates },
      });
    },
    [state.services, updateServices]
  );

  const updateProxy = useCallback(
    (updates: Partial<ProxyConfig> | null) => {
      if (updates === null) {
        updateServices({ ...state.services, proxy: null });
      } else {
        updateServices({
          ...state.services,
          proxy: {
            ...(state.services.proxy || {
              hostname: '',
              port: 8080,
              username: null,
              password: null,
            }),
            ...updates,
          },
        });
      }
    },
    [state.services, updateServices]
  );

  const addServer = useCallback(() => {
    const trimmed = newServer.trim();
    if (trimmed && !state.services.ntp.servers.includes(trimmed)) {
      updateNtp({ servers: [...state.services.ntp.servers, trimmed] });
      setNewServer('');
    }
  }, [newServer, state.services.ntp.servers, updateNtp]);

  const removeServer = useCallback(
    (server: string) => {
      updateNtp({
        servers: state.services.ntp.servers.filter(s => s !== server),
      });
    },
    [state.services.ntp.servers, updateNtp]
  );

  const ntp = state.services.ntp;
  const proxy = state.services.proxy;
  const showProxy = proxy !== null;

  const serverError = newServer.trim() ? validateNtpServer(newServer.trim()) : null;

  return (
    <View style={styles.container}>
      <Text variant="titleMedium" style={[styles.title, { color: theme.colors.onSurface }]}>
        Configure Services
      </Text>

      {/* NTP Section */}
      <Text variant="titleSmall" style={[styles.sectionTitle, { color: theme.colors.primary }]}>
        NTP (Time Synchronization)
      </Text>

      <View style={styles.ntpRow}>
        <Text variant="bodyMedium">Automatic time servers</Text>
        <Switch
          value={ntp.mode === 'automatic'}
          onValueChange={v => updateNtp({ mode: v ? 'automatic' : 'manual' })}
          accessibilityLabel="Automatic time servers"
        />
      </View>

      {ntp.mode === 'manual' && (
        <View style={styles.serverSection}>
          {ntp.servers.map(server => (
            <View key={server} style={styles.serverRow}>
              <Text variant="bodyMedium" style={styles.serverText}>
                {server}
              </Text>
              <IconButton
                icon="close"
                size={16}
                onPress={() => removeServer(server)}
                accessibilityLabel={`Remove server ${server}`}
              />
            </View>
          ))}

          <View style={styles.addServerRow}>
            <TextInput
              label="NTP Server"
              value={newServer}
              onChangeText={setNewServer}
              mode="outlined"
              autoCapitalize="none"
              placeholder="pool.ntp.org"
              style={styles.serverInput}
              accessibilityLabel="NTP server hostname or IP"
              error={!!serverError}
            />
            <IconButton
              icon="plus"
              mode="contained"
              onPress={addServer}
              disabled={!newServer.trim() || !!serverError}
              accessibilityLabel="Add NTP server"
            />
          </View>
          {serverError && (
            <HelperText type="error" visible={true}>
              {serverError}
            </HelperText>
          )}

          {ntp.servers.length === 0 && (
            <HelperText type="info" visible={true}>
              Add at least one NTP server
            </HelperText>
          )}
        </View>
      )}

      <Divider style={styles.divider} />

      {/* Proxy Section */}
      <View style={styles.proxyHeader}>
        <Text variant="titleSmall" style={{ color: theme.colors.primary }}>
          HTTP Proxy
        </Text>
        <Switch
          value={showProxy}
          onValueChange={v => {
            if (v) {
              updateProxy({ hostname: '', port: 8080, username: null, password: null });
            } else {
              updateProxy(null);
            }
          }}
          accessibilityLabel="Enable HTTP proxy"
        />
      </View>

      {showProxy && proxy && (
        <View style={styles.proxyFields}>
          <TextInput
            label="Proxy Hostname"
            value={proxy.hostname}
            onChangeText={v => updateProxy({ hostname: v })}
            mode="outlined"
            autoCapitalize="none"
            placeholder="proxy.example.com"
            accessibilityLabel="Proxy hostname"
            style={styles.input}
          />

          <TextInput
            label="Proxy Port"
            value={proxy.port.toString()}
            onChangeText={v => {
              const num = parseInt(v, 10);
              updateProxy({ port: isNaN(num) ? 0 : num });
            }}
            mode="outlined"
            keyboardType="number-pad"
            error={!!validatePort(proxy.port)}
            accessibilityLabel="Proxy port"
            style={styles.input}
          />
          {validatePort(proxy.port) && (
            <HelperText type="error" visible={true}>
              {validatePort(proxy.port)}
            </HelperText>
          )}

          <TextInput
            label="Username (optional)"
            value={proxy.username || ''}
            onChangeText={v => updateProxy({ username: v || null })}
            mode="outlined"
            autoCapitalize="none"
            accessibilityLabel="Proxy username"
            style={styles.input}
          />

          <TextInput
            label="Password (optional)"
            value={proxy.password || ''}
            onChangeText={v => updateProxy({ password: v || null })}
            mode="outlined"
            secureTextEntry
            accessibilityLabel="Proxy password"
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
    marginBottom: spacing.md,
  },
  sectionTitle: {
    marginBottom: spacing.xs,
  },
  ntpRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: spacing.sm,
  },
  serverSection: {
    paddingLeft: spacing.md,
    marginBottom: spacing.sm,
  },
  serverRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: spacing.xs,
  },
  serverText: {
    fontFamily: 'monospace',
  },
  addServerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
  },
  serverInput: {
    flex: 1,
  },
  divider: {
    marginVertical: spacing.md,
  },
  proxyHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: spacing.sm,
  },
  proxyFields: {
    paddingLeft: spacing.md,
  },
  input: {
    marginBottom: spacing.xs,
  },
});
