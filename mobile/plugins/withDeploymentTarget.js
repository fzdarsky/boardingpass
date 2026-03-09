/**
 * Expo config plugin to set iOS deployment target in Podfile.properties.json
 *
 * Ensures CocoaPods uses the same deployment target as the Xcode project,
 * preventing pod compatibility errors for packages requiring newer iOS versions.
 */

const { withDangerousMod } = require('@expo/config-plugins');
const fs = require('fs');
const path = require('path');

function withDeploymentTarget(config, { deploymentTarget }) {
  return withDangerousMod(config, [
    'ios',
    (config) => {
      const podfilePropsPath = path.join(
        config.modRequest.platformProjectRoot,
        'Podfile.properties.json'
      );

      let props = {};
      if (fs.existsSync(podfilePropsPath)) {
        props = JSON.parse(fs.readFileSync(podfilePropsPath, 'utf8'));
      }

      props['ios.deploymentTarget'] = deploymentTarget;
      fs.writeFileSync(podfilePropsPath, JSON.stringify(props, null, 2) + '\n');

      return config;
    },
  ]);
}

module.exports = withDeploymentTarget;
