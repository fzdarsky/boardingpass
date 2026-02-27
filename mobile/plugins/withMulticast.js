/**
 * Expo config plugin to add iOS multicast networking entitlement
 * Required for mDNS/Bonjour discovery on iOS 14+
 *
 * @see https://developer.apple.com/documentation/bundleresources/entitlements/com_apple_developer_networking_multicast
 */

const { withEntitlementsPlist } = require('@expo/config-plugins');

/**
 * Add multicast networking entitlement to iOS
 * This is required for react-native-zeroconf mDNS discovery to work on iOS 14+
 */
function withMulticast(config) {
  return withEntitlementsPlist(config, (config) => {
    // Add multicast networking entitlement
    config.modResults['com.apple.developer.networking.multicast'] = true;
    return config;
  });
}

module.exports = withMulticast;
