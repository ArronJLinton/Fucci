import Constants from 'expo-constants';
import {Platform} from 'react-native';

// Environment variables are available through Constants.expoConfig.extra
// We'll define them in app.json under the "extra" section

export type AppEnv = 'development' | 'staging' | 'production';

interface EnvironmentConfig {
  APP_ENV: AppEnv;
  API_BASE_URL: string;
  APP_NAME: string;
  APP_VERSION: string;
  NODE_ENV: 'development' | 'production' | 'test';
  DEBUG: boolean;
}

// Helper function to get the correct localhost host for development
const getLocalHost = (): string => {
  // Android emulator uses 10.0.2.2 to access the host machine
  // iOS simulator can use localhost
  return Platform.OS === 'android' ? '10.0.2.2' : 'localhost';
};

const defaultDevApi = 'http://localhost:8080/v1/api';

function normalizeAppEnv(raw: unknown): AppEnv {
  if (raw === 'staging' || raw === 'production' || raw === 'development') {
    return raw;
  }
  return 'development';
}

// Get environment variables from app.json extra (and app.config.js / EAS env merges)
const getEnvironmentConfig = (): EnvironmentConfig => {
  const extra = Constants.expoConfig?.extra || {};
  const appEnv = normalizeAppEnv(extra.APP_ENV);

  let baseURL: string;
  switch (appEnv) {
    case 'development':
      baseURL =
        (typeof extra.API_DEV_URL === 'string' && extra.API_DEV_URL) ||
        defaultDevApi;
      break;
    case 'staging':
      baseURL =
        (typeof extra.API_STAGING_URL === 'string' && extra.API_STAGING_URL) ||
        (typeof extra.API_BASE_URL === 'string' && extra.API_BASE_URL) ||
        defaultDevApi;
      break;
    default:
      baseURL =
        (typeof extra.API_BASE_URL === 'string' && extra.API_BASE_URL) ||
        defaultDevApi;
  }

  // Android emulator cannot reach host "localhost"; use 10.0.2.2
  if (baseURL.includes('localhost') && Platform.OS !== 'web') {
    const localHost = getLocalHost();
    baseURL = baseURL.replace('localhost', localHost);
  }

  return {
    APP_ENV: appEnv,
    API_BASE_URL: baseURL,
    APP_NAME: extra.APP_NAME || 'Fucci',
    APP_VERSION: extra.APP_VERSION || '1.0.0',
    NODE_ENV:
      (extra.NODE_ENV as EnvironmentConfig['NODE_ENV']) || 'development',
    DEBUG: extra.DEBUG === 'true' || extra.DEBUG === true,
  };
};

export const environment = getEnvironmentConfig();

// Log the API base URL when not production app env (local / staging debugging)
if (environment.APP_ENV !== 'production') {
  console.log(
    '[Environment] APP_ENV:',
    environment.APP_ENV,
    'API_BASE_URL:',
    environment.API_BASE_URL,
  );
}

// Helper functions for common environment checks
export const isDevelopment = () => environment.APP_ENV === 'development';
export const isStaging = () => environment.APP_ENV === 'staging';
export const isProduction = () => environment.APP_ENV === 'production';
export const isDebug = () => environment.DEBUG;

// API configuration
export const apiConfig = {
  baseURL: environment.API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
};

export default environment;
