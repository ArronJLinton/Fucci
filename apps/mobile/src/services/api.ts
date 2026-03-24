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
  if (!response.ok) {
    const errBody = await response.json().catch(() => ({}));
    const message =
      (typeof errBody.message === 'string' && errBody.message) ||
      (typeof errBody.error === 'string' && errBody.error) ||
      `Request failed: ${response.status}`;
    throw new ApiRequestError(message, response.status);
  }
  return response.json();
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
