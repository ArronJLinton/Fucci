import {apiConfig} from '../config/environment';
import {makeAuthRequest} from './api';

// Auth types (005 user registration) — email-only (no username)
export interface RegisterRequest {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  photo_url?: string;
}

export interface AuthUser {
  id: number;
  firstname: string;
  lastname: string;
  email: string;
  display_name?: string;
  avatar_url?: string;
  role?: string;
  created_at?: string;
}

export interface RegisterResponse {
  user: AuthUser;
  token: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  user: AuthUser;
  token: string;
}

export interface FollowingItem {
  id: string;
  type: string;
  followable_id: string;
  name?: string;
}

// POST /auth/register (email-only)
export const register = async (
  body: RegisterRequest,
): Promise<
  | {ok: true; data: RegisterResponse}
  | {
      ok: false;
      status: number;
      message: string;
      errors?: Array<{field: string; message: string}>;
    }
> => {
  const url = `${apiConfig.baseURL}/auth/register`;
  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: {...apiConfig.headers},
      body: JSON.stringify({
        firstname: body.first_name,
        lastname: body.last_name,
        email: body.email,
        password: body.password,
        avatar_url: body.photo_url || undefined,
      }),
    });
    const data = await response.json().catch(() => ({}));
    if (response.status === 201 && data.user && data.token) {
      return {ok: true, data: data as RegisterResponse};
    }
    const message =
      data.message || data.error || `Request failed (${response.status})`;
    const errors = data.errors;
    return {ok: false, status: response.status, message, errors};
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Network error';
    return {ok: false, status: 0, message};
  }
};

// POST /auth/login
export const login = async (
  body: LoginRequest,
): Promise<
  {ok: true; data: LoginResponse} | {ok: false; status: number; message: string}
> => {
  const url = `${apiConfig.baseURL}/auth/login`;
  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: {...apiConfig.headers},
      body: JSON.stringify({
        email: body.email,
        password: body.password,
      }),
    });
    const data = await response.json().catch(() => ({}));
    if (response.status === 200 && data.user && data.token) {
      return {ok: true, data: data as LoginResponse};
    }
    const message =
      data.message || data.error || `Request failed (${response.status})`;
    return {ok: false, status: response.status, message};
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Network error';
    return {ok: false, status: 0, message};
  }
};

// GET /users/profile (auth required)
export const getProfile = async (token: string): Promise<AuthUser | null> => {
  try {
    const data = await makeAuthRequest(token, '/users/profile', 'GET');
    return data as AuthUser;
  } catch {
    return null;
  }
};

// PUT /users/profile (auth required)
export const updateProfile = async (
  token: string,
  body: {
    firstname?: string;
    lastname?: string;
    display_name?: string;
    avatar_url?: string;
  },
): Promise<AuthUser> => {
  const data = await makeAuthRequest(token, '/users/profile', 'PUT', {
    body: JSON.stringify(body),
  });
  return data as AuthUser;
};

// GET /users/me/following (auth required)
export const getFollowing = async (token: string): Promise<FollowingItem[]> => {
  try {
    const data = await makeAuthRequest(token, '/users/me/following', 'GET');
    return (data?.items ?? []) as FollowingItem[];
  } catch {
    return [];
  }
};
