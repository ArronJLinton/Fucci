import {apiConfig} from '../config/environment';

/** Shown when the backend returns 5xx or an unauthenticated request fails with a server error. */
export const BACKEND_UNAVAILABLE_MESSAGE =
  'Something went wrong. Please refresh and try again. If the problem persists, reach out to support.';

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
    if (error.status === 401) {
      return 'Session expired. Please sign in again.';
    }
    // 403 is also returned by third-party proxies (e.g. RapidAPI key / subscription)
    if (error.status === 403) {
      if (
        error.message &&
        !/^Request failed \(403\)$/.test(error.message.trim())
      ) {
        return error.message;
      }
      return 'Access denied. If you’re the developer, verify RAPID_API_KEY and that this app is subscribed to the Snapchat API on RapidAPI.';
    }
    if (error.status === 408) {
      return 'Request timed out. Please try again.';
    }
    if (error.status === 429) {
      return 'Too many requests. Please wait and try again.';
    }
    if (error.status >= 500) {
      return BACKEND_UNAVAILABLE_MESSAGE;
    }
    return error.message;
  }
  if (error instanceof Error && error.message) {
    const m = error.message;
    // Legacy plain errors from fetch helpers (before ApiRequestError).
    if (/API request failed:\s*5\d\d/i.test(m)) {
      return BACKEND_UNAVAILABLE_MESSAGE;
    }
    if (
      /network|Unable to connect|ECONNRESET|ENOTFOUND|fetch failed/i.test(m) ||
      (error instanceof TypeError && /fetch/i.test(m))
    ) {
      return 'Network error. Check your connection and try again.';
    }
    return m;
  }
  return BACKEND_UNAVAILABLE_MESSAGE;
}

/**
 * `RequestInit.headers` may be a `Headers` instance, `[name, value][]`, or a plain
 * object; only the latter is safe to spread. Normalize so merges never drop
 * custom headers.
 */
function headersInitToRecord(
  init: HeadersInit | undefined,
): Record<string, string> {
  if (init == null) {
    return {};
  }
  const h = new Headers(init);
  const out: Record<string, string> = {};
  h.forEach((value, name) => {
    out[name] = value;
  });
  return out;
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
  const {headers: optionsHeaders, ...restOptions} = options;
  const url = `${apiConfig.baseURL}${endpoint}`;
  try {
    const response = await fetch(url, {
      ...restOptions,
      method,
      redirect: 'follow',
      headers: {
        ...apiConfig.headers,
        ...headersInitToRecord(optionsHeaders),
      },
    });
    if (!response.ok) {
      const text = await response.text();
      let errMsg = `Request failed (${response.status})`;
      if (text.trim()) {
        try {
          const errBody = JSON.parse(text) as Record<string, unknown>;
          const m =
            (typeof errBody.message === 'string' && errBody.message) ||
            (typeof errBody.error === 'string' && errBody.error);
          if (m) {
            errMsg = m;
          }
        } catch {
          if (text.length < 500) {
            errMsg = text.trim();
          }
        }
      }
      throw new ApiRequestError(errMsg, response.status);
    }
    if (response.status === 204 || response.status === 205) {
      return undefined;
    }
    if (response.headers.get('content-length') === '0') {
      return undefined;
    }
    try {
      return await response.json();
    } catch (e) {
      if (e instanceof SyntaxError) {
        const msg = (e instanceof Error ? e.message : String(e)).toLowerCase();
        if (
          /unexpected end|end of (data|input|json|the json input|file|stream)/.test(
            msg,
          ) ||
          msg.includes('unterminated input')
        ) {
          return undefined;
        }
        throw new ApiRequestError(
          BACKEND_UNAVAILABLE_MESSAGE,
          response.status,
        );
      }
      if (e instanceof TypeError) {
        throw new ApiRequestError(
          BACKEND_UNAVAILABLE_MESSAGE,
          response.status,
        );
      }
      throw e;
    }
  } catch (error) {
    if (error instanceof ApiRequestError) {
      throw error;
    }
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
      ...headersInitToRecord(optionsHeaders),
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
