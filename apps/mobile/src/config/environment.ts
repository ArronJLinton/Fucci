import Constants from 'expo-constants';
import {Platform} from 'react-native';

// Environment variables are available through Constants.expoConfig.extra
// We'll define them in app.json under the "extra" section

interface EnvironmentConfig {
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

// Get environment variables from app.json extra section
const getEnvironmentConfig = (): EnvironmentConfig => {
  const extra = Constants.expoConfig?.extra || {};
  // Get base URL from config or use default
  let baseURL = extra.API_BASE_URL || 'http://localhost:8080/v1/api';

  // Always replace localhost with platform-specific host when running on mobile
  // Android emulator cannot access localhost, needs 10.0.2.2
  // iOS simulator can use localhost, but we'll replace it for consistency
  // This is needed regardless of NODE_ENV because localhost doesn't work on emulators
  if (baseURL.includes('localhost') && Platform.OS !== 'web') {
    const localHost = getLocalHost();
    baseURL = baseURL.replace('localhost', localHost);
  }

  return {
    API_BASE_URL: baseURL,
    APP_NAME: extra.APP_NAME || 'Fucci',
    APP_VERSION: extra.APP_VERSION || '1.0.0',
    NODE_ENV:
      (extra.NODE_ENV as EnvironmentConfig['NODE_ENV']) || 'development',
    DEBUG: extra.DEBUG === 'true' || extra.DEBUG === true,
  };
};

export const environment = getEnvironmentConfig();

// Log the API base URL in development for debugging
if (environment.NODE_ENV === 'development') {
  console.log('[Environment] API_BASE_URL:', environment.API_BASE_URL);
}

// Helper functions for common environment checks
export const isDevelopment = () => environment.NODE_ENV === 'development';
export const isProduction = () => environment.NODE_ENV === 'production';
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
