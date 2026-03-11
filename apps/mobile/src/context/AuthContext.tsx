import React, {createContext, useCallback, useContext, useEffect, useState} from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';
import type {AuthUser} from '../services/api';

export const AUTH_TOKEN_KEY = '@fucci/auth_token';
export const AUTH_USER_KEY = '@fucci/auth_user';

interface AuthState {
  token: string | null;
  user: AuthUser | null;
  isReady: boolean;
}

interface AuthContextValue extends AuthState {
  isLoggedIn: boolean;
  setAuth: (token: string, user: AuthUser) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({children}: {children: React.ReactNode}) {
  const [state, setState] = useState<AuthState>({
    token: null,
    user: null,
    isReady: false,
  });

  const loadAuth = useCallback(async () => {
    try {
      const [token, userJson] = await Promise.all([
        AsyncStorage.getItem(AUTH_TOKEN_KEY),
        AsyncStorage.getItem(AUTH_USER_KEY),
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

  const setAuth = useCallback(async (token: string, user: AuthUser) => {
    await AsyncStorage.setItem(AUTH_TOKEN_KEY, token);
    await AsyncStorage.setItem(AUTH_USER_KEY, JSON.stringify(user));
    setState((s) => ({...s, token, user}));
  }, []);

  const logout = useCallback(async () => {
    await AsyncStorage.multiRemove([AUTH_TOKEN_KEY, AUTH_USER_KEY]);
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
