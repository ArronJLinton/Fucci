/**
 * Dynamic Expo config: merges app.json with build-time env (EAS profile, local shell).
 * Set APP_ENV=staging|production|development via eas.json "env" or when running set-env.js.
 *
 * API_DEV_URL / API_STAGING_URL / API_BASE_URL are derived from APP_ENV so EAS builds that
 * only set APP_ENV never ship with committed localhost URLs from app.json.
 * Override any URL with EXPO_PUBLIC_* or the same names without prefix (see set-env.js).
 */
const appJson = require('./app.json');

/** @param {...string | undefined} parts */
function firstString(...parts) {
  for (const p of parts) {
    if (typeof p === 'string' && p.trim() !== '') {
      return p.trim();
    }
  }
  return undefined;
}

module.exports = () => {
  const expo = {...appJson.expo};
  const extra = {...(expo.extra || {})};

  extra.APP_ENV =
    process.env.APP_ENV ||
    process.env.EXPO_PUBLIC_APP_ENV ||
    extra.APP_ENV ||
    'development';

  const appEnv = extra.APP_ENV;

  const defaultStaging =
    firstString(
      process.env.STAGING_API_BASE_URL,
      process.env.EXPO_PUBLIC_STAGING_API_BASE_URL,
    ) || 'https://fucci-api.fly.dev/v1/api';

  const defaultProduction =
    firstString(
      process.env.PRODUCTION_API_BASE_URL,
      process.env.EXPO_PUBLIC_PRODUCTION_API_BASE_URL,
    ) || 'https://fucci-api.fly.dev/v1/api';

  const defaultDev =
    firstString(
      process.env.DEV_API_BASE_URL,
      process.env.EXPO_PUBLIC_DEV_API_BASE_URL,
    ) || 'http://localhost:8080/v1/api';

  extra.API_STAGING_URL =
    firstString(
      process.env.API_STAGING_URL,
      process.env.EXPO_PUBLIC_API_STAGING_URL,
    ) || defaultStaging;

  extra.API_DEV_URL =
    firstString(
      process.env.API_DEV_URL,
      process.env.EXPO_PUBLIC_API_DEV_URL,
    ) || defaultDev;

  const explicitBase = firstString(
    process.env.API_BASE_URL,
    process.env.EXPO_PUBLIC_API_BASE_URL,
  );

  switch (appEnv) {
    case 'staging':
      // extra.API_STAGING_URL already folds STAGING_API_BASE_URL / API_STAGING_URL / default
      extra.API_BASE_URL = explicitBase || extra.API_STAGING_URL;
      break;
    case 'production':
      extra.API_BASE_URL = explicitBase || defaultProduction;
      break;
    default:
      extra.API_BASE_URL = explicitBase || extra.API_DEV_URL;
  }

  expo.extra = extra;
  return expo;
};
