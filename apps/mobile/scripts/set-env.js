#!/usr/bin/env node

/**
 * Environment setup script for Expo
 * Usage: node scripts/set-env.js [environment]
 *
 * Environments:
 * - development (default)
 * - production
 * - staging
 */

const fs = require('fs');
const path = require('path');

const defaultStagingApi =
  process.env.STAGING_API_BASE_URL ||
  'https://fucci-api.fly.dev/v1/api';
const defaultProductionApi =
  process.env.PRODUCTION_API_BASE_URL ||
  'https://fucci-api.fly.dev/v1/api';

const environments = {
  development: {
    APP_ENV: 'development',
    API_DEV_URL: 'http://localhost:8080/v1/api',
    API_BASE_URL: 'http://localhost:8080/v1/api',
    API_STAGING_URL: defaultStagingApi,
    APP_NAME: 'Fucci',
    NODE_ENV: 'development',
    DEBUG: 'true',
  },
  staging: {
    APP_ENV: 'staging',
    API_STAGING_URL: defaultStagingApi,
    API_BASE_URL: defaultStagingApi,
    APP_NAME: 'Fucci Staging',
    NODE_ENV: 'development',
    DEBUG: 'true',
  },
  production: {
    APP_ENV: 'production',
    API_BASE_URL: defaultProductionApi,
    API_STAGING_URL: defaultStagingApi,
    APP_NAME: 'Fucci',
    NODE_ENV: 'production',
    DEBUG: 'false',
  },
};

const targetEnv = process.argv[2] || 'development';

if (!environments[targetEnv]) {
  console.error(`❌ Unknown environment: ${targetEnv}`);
  console.log('Available environments:', Object.keys(environments).join(', '));
  process.exit(1);
}

// Update app.json
const appJsonPath = path.join(__dirname, '..', 'app.json');
const appJson = JSON.parse(fs.readFileSync(appJsonPath, 'utf8'));
const existingExtra = appJson.expo.extra || {};
const derivedAppVersion = String(appJson.expo.version || '1.0.0');
const envConfig = {
  ...environments[targetEnv],
  APP_VERSION: derivedAppVersion,
};

appJson.expo.extra = {
  ...existingExtra,
  ...envConfig,
};

fs.writeFileSync(appJsonPath, JSON.stringify(appJson, null, 2));

// Create .env file
const envContent = Object.entries(envConfig)
  .map(([key, value]) => `${key}=${value}`)
  .join('\n');

fs.writeFileSync(path.join(__dirname, '..', '.env'), envContent);

console.log(`✅ Environment set to: ${targetEnv}`);
console.log('Configuration:');
Object.entries(envConfig).forEach(([key, value]) => {
  console.log(`  ${key}: ${value}`);
});
