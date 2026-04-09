/**
 * Google sign-in is initiated via the API (GET /auth/google/start).
 * OAuth client credentials are not bundled in the app.
 */

export const GOOGLE_REDIRECT_URIS = {
  ios: 'fucci://auth',
  android: 'com.fucci.app:/oauth2redirect',
} as const;
