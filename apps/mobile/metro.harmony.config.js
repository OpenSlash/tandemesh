const { getDefaultConfig, mergeConfig } = require('@react-native/metro-config');
const path = require('path');
const { createHarmonyMetroConfig } = require('@react-native-oh/react-native-harmony/metro.config');

/**
 * Metro configuration for HarmonyOS / RNOH.
 * Android and iOS use metro.config.js.
 *
 * @type {import('@react-native/metro-config').MetroConfig}
 */
const config = {
  resolve: {
    extraNodeModules: {
      'react-native-svg': path.resolve(__dirname, 'node_modules/@react-native-ohos/react-native-svg'),
    },
  },
  transformer: {
    getTransformOptions: async () => ({
      transform: {
        experimentalImportSupport: false,
        inlineRequires: true,
      },
    }),
  },
};

module.exports = mergeConfig(
  getDefaultConfig(__dirname),
  createHarmonyMetroConfig({
    reactNativeHarmonyPackageName: '@react-native-oh/react-native-harmony',
  }),
  config,
);
