import React, {createContext, useCallback, useContext, useEffect, useState} from 'react';
import {Platform} from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import * as SecureStore from 'expo-secure-store';
import type {AuthUser} from '../services/api';

export const AUTH_TOKEN_KEY = 'fucci_auth_token';
export const AUTH_USER_KEY = 'fucci_auth_user';

const useSecureStorage = Platform.OS !== 'web';

interface AuthState {
  token: string | null;
  user: AuthUser | null;
  isReady: boolean;
}

interface AuthContextValue extends AuthState {
  isLoggedIn: boolean;
  /** Set auth state. When persist is false (session-only), auth is not written to storage and is cleared on next app launch. */
  setAuth: (token: string, user: AuthUser, persist?: boolean) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

async function getStoredToken(): Promise<string | null> {
  if (useSecureStorage) {
    return await SecureStore.getItemAsync(AUTH_TOKEN_KEY);
  }
  return await AsyncStorage.getItem(AUTH_TOKEN_KEY);
}

async function getStoredUser(): Promise<string | null> {
  if (useSecureStorage) {
    return await SecureStore.getItemAsync(AUTH_USER_KEY);
  }
  return await AsyncStorage.getItem(AUTH_USER_KEY);
}

async function setStoredAuth(token: string, userJson: string): Promise<void> {
  if (useSecureStorage) {
    await SecureStore.setItemAsync(AUTH_TOKEN_KEY, token);
    await SecureStore.setItemAsync(AUTH_USER_KEY, userJson);
  } else {
    await AsyncStorage.setItem(AUTH_TOKEN_KEY, token);
    await AsyncStorage.setItem(AUTH_USER_KEY, userJson);
  }
}

async function clearStoredAuth(): Promise<void> {
  if (useSecureStorage) {
    await SecureStore.deleteItemAsync(AUTH_TOKEN_KEY);
    await SecureStore.deleteItemAsync(AUTH_USER_KEY);
  } else {
    await AsyncStorage.multiRemove([AUTH_TOKEN_KEY, AUTH_USER_KEY]);
  }
}

export function AuthProvider({children}: {children: React.ReactNode}) {
  const [state, setState] = useState<AuthState>({
    token: null,
    user: null,
    isReady: false,
  });

  const loadAuth = useCallback(async () => {
    try {
      const [token, userJson] = await Promise.all([
        getStoredToken(),
        getStoredUser(),
      ]);
      let user: AuthUser | null = null;
      if (userJson) {
        try {
          user = JSON.parse(userJson) as AuthUser;
        } catch {
          // ignore invalid stored user
        }
      }
      setState({token, user, isReady: true});
    } catch {
      setState({token: null, user: null, isReady: true});
    }
  }, []);

  useEffect(() => {
    loadAuth();
  }, [loadAuth]);

  const setAuth = useCallback(async (token: string, user: AuthUser, persist = true) => {
    if (persist) {
      await setStoredAuth(token, JSON.stringify(user));
    } else {
      await clearStoredAuth();
    }
    setState((s) => ({...s, token, user}));
  }, []);

  const logout = useCallback(async () => {
    await clearStoredAuth();
    setState((s) => ({...s, token: null, user: null}));
  }, []);

  const value: AuthContextValue = {
    ...state,
    isLoggedIn: !!state.token,
    setAuth,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (ctx == null) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return ctx;
}
