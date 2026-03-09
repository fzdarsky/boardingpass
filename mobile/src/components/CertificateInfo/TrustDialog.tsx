/**
 * Certificate Trust Dialog Component
 *
 * Dialog for prompting user to trust a certificate (TOFU - Trust On First Use).
 * Displays certificate details and asks for user confirmation.
 */

import React, { useState } from 'react';
import { View, StyleSheet, ScrollView } from 'react-native';
import { Dialog, Portal, Button, Text, Icon, useTheme } from 'react-native-paper';
import { CertificateInfo } from '../../types/certificate';
import { CertificateInfoDisplay } from './index';

export interface CertificateTrustDialogProps {
  visible: boolean;
  certificate: CertificateInfo;
  onTrust: () => void;
  onReject: () => void;
  isChanged?: boolean;
}

/**
 * CertificateTrustDialog
 *
 * Displays a dialog for certificate trust decisions:
 * - New self-signed certificate: Asks user to confirm trust
 * - Changed certificate: Warns user and asks to confirm or reject
 */
export function CertificateTrustDialog({
  visible,
  certificate,
  onTrust,
  onReject,
  isChanged = false,
}: CertificateTrustDialogProps) {
  const theme = useTheme();
  const [accepting, setAccepting] = useState(false);

  const handleTrust = () => {
    setAccepting(true);
    onTrust();
  };

  const handleReject = () => {
    onReject();
  };

  // Determine dialog content based on certificate status
  const isNewSelfSigned = certificate.trustStatus === 'self_signed_new';
  const isCertChanged = certificate.trustStatus === 'changed' || isChanged;

  return (
    <Portal>
      <Dialog visible={visible} onDismiss={handleReject} style={styles.dialog}>
        <Dialog.Title style={styles.title}>
          {isCertChanged ? (
            <View style={styles.titleRow}>
              <Icon source="alert-circle" size={24} color={theme.colors.error} />
              <Text variant="titleLarge" style={{ color: theme.colors.error }}>
                Certificate Changed
              </Text>
            </View>
          ) : (
            <View style={styles.titleRow}>
              <Icon source="shield-alert" size={24} color={theme.colors.primary} />
              <Text variant="titleLarge">Trust Certificate?</Text>
            </View>
          )}
        </Dialog.Title>

        <Dialog.ScrollArea style={styles.scrollArea}>
          <ScrollView>
            {/* Warning message */}
            {isCertChanged && (
              <View style={[styles.warningBox, { backgroundColor: theme.colors.errorContainer }]}>
                <Icon source="alert" size={20} color={theme.colors.error} />
                <Text
                  variant="bodyMedium"
                  style={[styles.warningText, { color: theme.colors.error }]}
                >
                  The certificate for this device has changed since your last connection. This could
                  indicate a security issue or that the device was reconfigured.
                </Text>
              </View>
            )}

            {isNewSelfSigned && !isCertChanged && (
              <View style={[styles.infoBox, { backgroundColor: theme.colors.secondaryContainer }]}>
                <Icon source="information" size={20} color={theme.colors.secondary} />
                <Text variant="bodyMedium" style={styles.infoText}>
                  This device uses a self-signed certificate. You should verify the certificate
                  fingerprint matches the device before trusting it.
                </Text>
              </View>
            )}

            {/* Certificate Details */}
            <CertificateInfoDisplay certificate={certificate} compact={false} />
          </ScrollView>
        </Dialog.ScrollArea>

        <Dialog.Actions style={styles.actions}>
          <Button mode="outlined" onPress={handleReject} disabled={accepting} style={styles.button}>
            {isCertChanged ? 'Reject' : 'Cancel'}
          </Button>
          <Button
            mode="contained"
            onPress={handleTrust}
            loading={accepting}
            disabled={accepting}
            style={styles.button}
            buttonColor={isCertChanged ? theme.colors.error : theme.colors.primary}
          >
            {isCertChanged ? 'Trust Anyway' : 'Trust Certificate'}
          </Button>
        </Dialog.Actions>
      </Dialog>
    </Portal>
  );
}

const styles = StyleSheet.create({
  dialog: {
    maxHeight: '90%',
  },
  title: {
    paddingTop: 24,
  },
  titleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  scrollArea: {
    maxHeight: 500,
    paddingHorizontal: 0,
  },
  warningBox: {
    flexDirection: 'row',
    padding: 16,
    borderRadius: 8,
    marginBottom: 16,
    gap: 12,
  },
  warningText: {
    flex: 1,
    fontWeight: '600',
  },
  infoBox: {
    flexDirection: 'row',
    padding: 16,
    borderRadius: 8,
    marginBottom: 16,
    gap: 12,
  },
  infoText: {
    flex: 1,
  },
  actions: {
    paddingHorizontal: 24,
    paddingBottom: 16,
  },
  button: {
    marginLeft: 8,
  },
});
