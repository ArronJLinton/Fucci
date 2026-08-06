import {Platform} from 'react-native';
import * as AppleAuthentication from 'expo-apple-authentication';

import {apiConfig} from '../config/environment';
import type {AuthUser} from './auth';
import {resolvePostGoogleAuthRoute, type PostGoogleAuthRoute} from './googleAuth';

export type AppleAuthResult =
  | {kind: 'success'; token: string; user: AuthUser; isNew: boolean}
  | {kind: 'cancel'}
  | {kind: 'unavailable'}
  | {kind: 'error'; message: string};

export type AppleAuthAPIResponse = {
  token: string;
  user: AuthUser;
  is_new: boolean;
};

export async function isAppleAuthAvailable(): Promise<boolean> {
  if (Platform.OS !== 'ios') {
    return false;
  }
  try {
    return await AppleAuthentication.isAvailableAsync();
  } catch {
    return false;
  }
}

/**
 * Native Sign in with Apple → POST /auth/apple with verified identity token on the server.
 */
export async function launchAppleSignIn(): Promise<AppleAuthResult> {
  if (Platform.OS !== 'ios') {
    return {kind: 'unavailable'};
  }
  const available = await isAppleAuthAvailable();
  if (!available) {
    return {kind: 'unavailable'};
  }

  try {
    const credential = await AppleAuthentication.signInAsync({
      requestedScopes: [
        AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
        AppleAuthentication.AppleAuthenticationScope.EMAIL,
      ],
    });

    if (!credential.identityToken) {
      return {
        kind: 'error',
        message: 'Apple Sign-In did not return an identity token.',
      };
    }

    const fullName =
      credential.fullName &&
      (credential.fullName.givenName || credential.fullName.familyName)
        ? {
            given_name: credential.fullName.givenName ?? '',
            family_name: credential.fullName.familyName ?? '',
          }
        : undefined;

    const url = `${apiConfig.baseURL}/auth/apple`;
    const response = await fetch(url, {
      method: 'POST',
      headers: {...apiConfig.headers},
      body: JSON.stringify({
        identity_token: credential.identityToken,
        authorization_code: credential.authorizationCode ?? undefined,
        full_name: fullName,
      }),
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      const message =
        (typeof data.error === 'string' && data.error) ||
        (typeof data.message === 'string' && data.message) ||
        `Request failed (${response.status})`;
      return {kind: 'error', message};
    }
    if (typeof data.token !== 'string' || !data.user) {
      return {kind: 'error', message: 'Apple sign-in failed'};
    }
    const parsed = data as AppleAuthAPIResponse;
    return {
      kind: 'success',
      token: parsed.token,
      user: parsed.user,
      isNew: Boolean(parsed.is_new),
    };
  } catch (e: unknown) {
    const code =
      e && typeof e === 'object' && 'code' in e
        ? String((e as {code?: string}).code)
        : '';
    if (code === 'ERR_REQUEST_CANCELED') {
      return {kind: 'cancel'};
    }
    return {
      kind: 'error',
      message: e instanceof Error ? e.message : 'Apple sign-in failed',
    };
  }
}

export function resolvePostAppleAuthRoute(isNew: boolean): PostGoogleAuthRoute {
  return resolvePostGoogleAuthRoute(isNew);
}
