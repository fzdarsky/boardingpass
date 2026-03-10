/**
 * Dynamic Expo configuration
 *
 * This file allows conditional configuration based on environment variables.
 * It takes precedence over app.json when both exist.
 *
 * Environment Variables:
 *   APP_VERSION             - Override the app version (e.g., "0.1.0"). Set automatically
 *                             by CI from the git tag (app-v0.1.0 -> 0.1.0).
 *                             Default: "0.1.0"
 *   ENABLE_MDNS_ENTITLEMENT - Set to "false" to disable the multicast networking
 *                             entitlement (requires paid Apple Developer account).
 *                             Default: "true"
 */

// App version from environment (set by CI from git tag) or default
const appVersion = process.env.APP_VERSION || '0.1.0';

// Check if mDNS entitlement should be enabled (default: true)
const enableMdnsEntitlement = process.env.ENABLE_MDNS_ENTITLEMENT !== 'false';

// Base plugins that don't require special entitlements
const basePlugins = [
  'expo-router',
  [
    'expo-camera',
    {
      cameraPermission:
        'Camera access is required to scan QR codes for device connection',
      microphonePermission: false,
    },
  ],
  [
    'expo-barcode-scanner',
    {
      cameraPermission:
        'Camera access is required to scan QR codes for device connection',
    },
  ],
  ['./plugins/withDeploymentTarget.js', { deploymentTarget: '15.0' }],
];

// Conditionally add multicast plugin (requires paid Apple Developer account)
const plugins = enableMdnsEntitlement
  ? [...basePlugins, './plugins/withMulticast.js']
  : basePlugins;

if (!enableMdnsEntitlement) {
  console.log(
    '[app.config.js] mDNS entitlement disabled (ENABLE_MDNS_ENTITLEMENT=false)'
  );
  console.log(
    '[app.config.js] Automatic device discovery will not work on iOS'
  );
}

module.exports = {
  expo: {
    name: 'BoardingPass',
    slug: 'boardingpass-mobile',
    version: appVersion,
    orientation: 'portrait',
    icon: './assets/icon.png',
    splash: {
      image: './assets/splash.png',
      resizeMode: 'contain',
      backgroundColor: '#f3efe3',
    },
    userInterfaceStyle: 'automatic',
    scheme: 'boardingpass',
    assetBundlePatterns: ['**/*'],
    ios: {
      supportsTablet: true,
      bundleIdentifier: 'org.boardingpass-project.mobile',
      deploymentTarget: '15.0',
      infoPlist: {
        NSCameraUsageDescription:
          'Camera access is required to scan QR codes for device connection',
        NSLocalNetworkUsageDescription:
          'Network access is required to discover BoardingPass devices on your local network',
        NSBluetoothAlwaysUsageDescription:
          'Bluetooth is used to discover nearby BoardingPass devices',
        NSBonjourServices: ['_boardingpass._tcp'],
        ITSAppUsesNonExemptEncryption: false,
      },
      entitlements: {},
    },
    android: {
      package: 'org.boardingpass_project.mobile',
      compileSdkVersion: 34,
      targetSdkVersion: 34,
      minSdkVersion: 29,
      permissions: [
        'CAMERA',
        'ACCESS_FINE_LOCATION',
        'CHANGE_WIFI_MULTICAST_STATE',
        'INTERNET',
        'ACCESS_NETWORK_STATE',
        'BLUETOOTH_SCAN',
        'BLUETOOTH_CONNECT',
      ],
    },
    web: {
      bundler: 'metro',
    },
    plugins,
    experiments: {
      typedRoutes: true,
    },
    extra: {
      router: {
        origin: false,
      },
      eas: {
        projectId: '6a70081d-f13f-43e1-9cef-1600d138c47d',
      },
    },
  },
};
