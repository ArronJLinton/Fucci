import {apiConfig} from '../config/environment';

/** Thrown when an authenticated request returns a non-2xx status; includes HTTP status for callers. */
export class ApiRequestError extends Error {
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiRequestError';
    this.status = status;
    Object.setPrototypeOf(this, ApiRequestError.prototype);
  }
}

/** Short message for alerts / inline errors; uses status when available. */
export function userFacingApiMessage(error: unknown): string {
  if (error instanceof ApiRequestError) {
    if (error.status === 401 || error.status === 403) {
      return 'Session expired. Please sign in again.';
    }
    if (error.status === 408 || error.status === 429) {
      return 'Too many requests. Please wait and try again.';
    }
    if (error.status >= 500) {
      return 'Server error. Please try again in a moment.';
    }
    return error.message;
  }
  if (error instanceof Error && error.message) {
    return /network|fetch|failed/i.test(error.message)
      ? 'Network error. Check your connection and try again.'
      : error.message;
  }
  return 'Something went wrong. Please try again.';
}

/**
 * Unauthenticated API request helper.
 * Used by futbol and debate modules; also re-exported for any direct callers.
 */
export const makeApiRequest = async (
  endpoint: string,
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH' = 'GET',
  options: RequestInit = {},
) => {
  const url = `${apiConfig.baseURL}${endpoint}`;
  try {
    const response = await fetch(url, {
      method,
      redirect: 'follow',
      headers: {
        ...apiConfig.headers,
        ...options.headers,
      },
      ...options,
    });
    if (!response.ok) {
      throw new Error(
        `API request failed: ${response.status} ${response.statusText}`,
      );
    }
    return response.json();
  } catch (error) {
    console.error(`API request failed for ${url}:`, error);
    throw error;
  }
};

/**
 * Authenticated API request helper (adds Bearer token).
 * Used by auth and debate modules.
 */
export const makeAuthRequest = async (
  token: string,
  endpoint: string,
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH' = 'GET',
  options: RequestInit = {},
) => {
  const {headers: optionsHeaders, ...restOptions} = options;
  const url = `${apiConfig.baseURL}${endpoint}`;
  const response = await fetch(url, {
    ...restOptions,
    method,
    headers: {
      ...apiConfig.headers,
      Authorization: `Bearer ${token}`,
      ...(optionsHeaders && typeof optionsHeaders === 'object'
        ? optionsHeaders
        : {}),
    },
  });
  const text = await response.text();
  if (!response.ok) {
    let errBody: Record<string, unknown> = {};
    if (text.trim()) {
      try {
        errBody = JSON.parse(text) as Record<string, unknown>;
      } catch {
        /* non-JSON error body */
      }
    }
    const message =
      (typeof errBody.message === 'string' && errBody.message) ||
      (typeof errBody.error === 'string' && errBody.error) ||
      (text.trim() && Object.keys(errBody).length === 0 ? text.trim() : '') ||
      `Request failed: ${response.status}`;
    throw new ApiRequestError(message, response.status);
  }
  if (!text.trim()) {
    return undefined;
  }
  try {
    return JSON.parse(text);
  } catch {
    throw new ApiRequestError('Invalid JSON in response', response.status);
  }
};

// Re-export auth module (register, login, getProfile, updateProfile, getFollowing, types)
export {
  register,
  login,
  getProfile,
  updateProfile,
  getFollowing,
  type RegisterRequest,
  type RegisterResponse,
  type LoginRequest,
  type LoginResponse,
  type AuthUser,
  type FollowingItem,
} from './auth';

// Re-export futbol module
export {
  fetchMatches,
  fetchLineup,
  fetchStandings,
  createMatch,
  updateMatch,
  deleteMatch,
  type Standing,
  type Match,
  type LineupData,
} from './futbol';

// Re-export debate module
export {
  fetchDebatesByMatch,
  fetchDebatesPublicFeed,
  fetchDebatesFeed,
  generateDebateSet,
  fetchDebateById,
  setCardVote,
  listComments,
  createComment,
  setCommentVote,
  addCommentReaction,
  removeCommentReaction,
  createDebate,
  fetchDebate,
  type GenerateDebateSetResponse,
} from './debate';
