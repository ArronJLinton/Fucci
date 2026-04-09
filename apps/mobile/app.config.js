/**
 * Dynamic Expo config: merges app.json with build-time env (EAS profile, local shell).
 * Set APP_ENV=staging|production|development via eas.json "env" or when running set-env.js.
 */
const appJson = require('./app.json');

module.exports = () => {
  const expo = {...appJson.expo};
  const extra = {...(expo.extra || {})};
  extra.APP_ENV =
    process.env.APP_ENV ||
    process.env.EXPO_PUBLIC_APP_ENV ||
    extra.APP_ENV ||
    'development';
  expo.extra = extra;
  return expo;
};
