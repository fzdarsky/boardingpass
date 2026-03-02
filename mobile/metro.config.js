const { getDefaultConfig } = require('expo/metro-config');
const path = require('path');

const config = getDefaultConfig(__dirname);

// Support for monorepo structure
const projectRoot = __dirname;
const workspaceRoot = path.resolve(projectRoot, '..');

config.watchFolders = [workspaceRoot];
config.resolver.nodeModulesPaths = [
  path.resolve(projectRoot, 'node_modules'),
  path.resolve(workspaceRoot, 'node_modules'),
];

// Ensure we resolve packages from mobile/node_modules first
config.resolver.disableHierarchicalLookup = false;

// Configure resolver to use React Native and browser-compatible modules
// This is critical for packages like axios and secure-remote-password that have
// both Node.js and browser/react-native versions
config.resolver.resolverMainFields = ['react-native', 'browser', 'main'];

// Resolve package exports field (for modern packages using "exports" in package.json)
config.resolver.unstable_enablePackageExports = true;

module.exports = config;
