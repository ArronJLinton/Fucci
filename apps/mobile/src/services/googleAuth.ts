import {Platform} from 'react-native';
import * as AuthSession from 'expo-auth-session';

import {
  GOOGLE_AUTH_SCOPES,
  GOOGLE_OAUTH_CLIENT_ID,
  GOOGLE_REDIRECT_URIS,
} from '../config/googleAuth';

export interface GoogleAuthCodeResult {
  code: string;
  redirectUri: string;
}

export async function launchGoogleAuthCodeFlow(): Promise<GoogleAuthCodeResult | null> {
  const redirectUri =
    Platform.OS === 'android'
      ? GOOGLE_REDIRECT_URIS.android
      : GOOGLE_REDIRECT_URIS.ios;

  const authUrl = new URL('https://accounts.google.com/o/oauth2/v2/auth');
  authUrl.searchParams.set('client_id', GOOGLE_OAUTH_CLIENT_ID);
  authUrl.searchParams.set('response_type', 'code');
  authUrl.searchParams.set('scope', GOOGLE_AUTH_SCOPES.join(' '));
  authUrl.searchParams.set('redirect_uri', redirectUri);

  const result = await AuthSession.startAsync({authUrl: authUrl.toString()});
  if (result.type !== 'success' || !result.params?.code) {
    return null;
  }

  return {
    code: String(result.params.code),
    redirectUri,
  };
}

