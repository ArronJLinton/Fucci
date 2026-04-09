import * as Linking from 'expo-linking';
import * as WebBrowser from 'expo-web-browser';

import {apiConfig} from '../config/environment';

export type PostGoogleAuthRoute = 'CreatePlayerProfile' | 'Main';

export type GoogleBrowserAuthResult =
  | {kind: 'success'; token: string; isNew: boolean}
  | {kind: 'cancel'}
  | {kind: 'error'; message: string};

function firstQuery(
  v: string | string[] | undefined,
): string | undefined {
  if (v == null) {
    return undefined;
  }
  return Array.isArray(v) ? v[0] : v;
}

/**
 * Opens the backend-driven Google OAuth flow: GET /auth/google/start → Google →
 * GET /auth/google/callback → redirect to this app with ?token=…&is_new=….
 * Client ID and secret stay on the API only.
 */
export async function launchGoogleAuthBrowserFlow(): Promise<GoogleBrowserAuthResult> {
  WebBrowser.maybeCompleteAuthSession();

  const returnUrl = Linking.createURL('auth');
  const startUrl = `${apiConfig.baseURL}/auth/google/start?return=${encodeURIComponent(returnUrl)}`;

  const result = await WebBrowser.openAuthSessionAsync(startUrl, returnUrl);

  if (result.type !== 'success' || !result.url) {
    return {kind: 'cancel'};
  }

  const parsed = Linking.parse(result.url);
  const q = parsed.queryParams ?? {};

  const googleErr = firstQuery(q.google_error);
  if (googleErr != null && googleErr !== '') {
    const desc = firstQuery(q.google_error_description);
    return {
      kind: 'error',
      message: desc || googleErr || 'Google sign-in failed',
    };
  }

  const token = firstQuery(q.token);
  if (!token) {
    return {kind: 'cancel'};
  }

  const rawNew = firstQuery(q.is_new);
  const isNew = rawNew === '1' || rawNew === 'true';

  return {kind: 'success', token, isNew};
}

export function resolvePostGoogleAuthRoute(isNew: boolean): PostGoogleAuthRoute {
  return isNew ? 'CreatePlayerProfile' : 'Main';
}
