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

const environments = {
  development: {
    API_BASE_URL: 'http://localhost:8080/v1/api',
    APP_NAME: 'Fucci',
    APP_VERSION: '1.0.0',
    NODE_ENV: 'development',
    DEBUG: 'true',
  },
  staging: {
    API_BASE_URL: 'https://fucci-api.fly.dev/v1/api',
    APP_NAME: 'Fucci Staging',
    APP_VERSION: '1.0.0',
    NODE_ENV: 'development',
    DEBUG: 'true',
  },
  production: {
    API_BASE_URL: 'https://fucci-api.fly.dev/v1/api',
    APP_NAME: 'Fucci',
    APP_VERSION: '1.0.0',
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

const envConfig = environments[targetEnv];

// Update app.json
const appJsonPath = path.join(__dirname, '..', 'app.json');
const appJson = JSON.parse(fs.readFileSync(appJsonPath, 'utf8'));
const existingExtra = appJson.expo.extra || {};

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
