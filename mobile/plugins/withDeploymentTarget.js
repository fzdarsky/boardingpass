/**
 * Expo config plugin to set iOS deployment target consistently across the project.
 *
 * Updates both:
 * - Podfile.properties.json (for CocoaPods platform version)
 * - Xcode project build settings (IPHONEOS_DEPLOYMENT_TARGET)
 *
 * This prevents linker warnings about objects built for newer iOS versions
 * than the project target.
 */

const { withDangerousMod, withXcodeProject } = require('@expo/config-plugins');
const fs = require('fs');
const path = require('path');

function withDeploymentTarget(config, { deploymentTarget }) {
  // 1. Set deployment target in Podfile.properties.json for CocoaPods
  config = withDangerousMod(config, [
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

  // 2. Set IPHONEOS_DEPLOYMENT_TARGET in the Xcode project build settings
  config = withXcodeProject(config, (config) => {
    const project = config.modResults;
    const configurations = project.pbxXCBuildConfigurationSection();

    for (const key in configurations) {
      const buildSettings = configurations[key].buildSettings;
      if (buildSettings && buildSettings.IPHONEOS_DEPLOYMENT_TARGET) {
        buildSettings.IPHONEOS_DEPLOYMENT_TARGET = deploymentTarget;
      }
    }

    return config;
  });

  return config;
}

module.exports = withDeploymentTarget;
