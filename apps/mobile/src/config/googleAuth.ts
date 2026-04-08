import Constants from 'expo-constants';

const extra = Constants.expoConfig?.extra ?? {};

export const GOOGLE_AUTH_SCOPES = ['openid', 'email', 'profile'] as const;

export const GOOGLE_REDIRECT_URIS = {
  ios: 'fucci://auth',
  android: 'com.fucci.app:/oauth2redirect',
} as const;

export const GOOGLE_OAUTH_CLIENT_ID =
  (extra.GOOGLE_OAUTH_CLIENT_ID as string | undefined) ?? '';

