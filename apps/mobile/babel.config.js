/** @type {import('@babel/core').ConfigFunction} */
module.exports = function (api) {
  api.cache(true);
  return {
    presets: ['babel-preset-expo'],
    // Reanimated plugin must be last (https://docs.swmansion.com/react-native-reanimated/docs/fundamentals/getting-started/)
    plugins: ['react-native-reanimated/plugin'],
  };
};
