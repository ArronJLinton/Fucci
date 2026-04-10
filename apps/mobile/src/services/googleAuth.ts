import * as Linking from 'expo-linking';
import * as WebBrowser from 'expo-web-browser';

import {apiConfig} from '../config/environment';
import type {GoogleAuthResponse} from './auth';

export type PostGoogleAuthRoute = 'CreatePlayerProfile' | 'Main';

export type GoogleBrowserAuthResult =
  | {kind: 'success'; token: string; user: GoogleAuthResponse['user']; isNew: boolean}
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

async function exchangeGoogleOAuthCode(
  code: string,
): Promise<
  | {ok: true; data: GoogleAuthResponse}
  | {ok: false; message: string}
> {
  const url = `${apiConfig.baseURL}/auth/google/exchange`;
  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: {...apiConfig.headers},
      body: JSON.stringify({code}),
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      const message =
        (typeof data.error === 'string' && data.error) ||
        (typeof data.message === 'string' && data.message) ||
        `Request failed (${response.status})`;
      return {ok: false, message};
    }
    if (typeof data.token !== 'string' || !data.user) {
      return {ok: false, message: 'Google sign-in failed'};
    }
    return {ok: true, data: data as GoogleAuthResponse};
  } catch (error) {
    return {
      ok: false,
      message:
        error instanceof Error ? error.message : 'Google sign-in failed',
    };
  }
}

/**
 * Opens the backend-driven Google OAuth flow: GET /auth/google/start → Google →
 * GET /auth/google/callback → redirect to this app with ?code=…&is_new=….
 * The app then exchanges the short-lived code at POST /auth/google/exchange.
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

  const exchangeCode = firstQuery(q.code);
  if (!exchangeCode) {
    return {kind: 'cancel'};
  }

  const exchanged = await exchangeGoogleOAuthCode(exchangeCode);
  if (!exchanged.ok) {
    return {kind: 'error', message: exchanged.message};
  }

  return {
    kind: 'success',
    token: exchanged.data.token,
    user: exchanged.data.user,
    isNew: exchanged.data.is_new,
  };
}

export function resolvePostGoogleAuthRoute(isNew: boolean): PostGoogleAuthRoute {
  return isNew ? 'CreatePlayerProfile' : 'Main';
}
