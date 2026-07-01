import type {Match} from '../types/match';
import type {DebateResponse} from '../types/debate';

export type PushNotificationData = {
  type?: 'debate' | 'match' | 'news';
  route?: string;
  params?: {
    debateId?: number;
    matchId?: number | string;
    url?: string;
    shortVideoId?: string;
  };
};

export type PushNavigationTarget =
  | {kind: 'news'; url: string}
  | {kind: 'debate'; match: Match; debate: DebateResponse}
  | {kind: 'match'; match: Match}
  | {kind: 'debates_tab'}
  | {kind: 'home_tab'};

export type PushPrefetchContext = {
  debate?: DebateResponse;
  match?: Match;
};

/** Normalize Expo notification `data` (nested or flat params). */
export function normalizePushNotificationData(
  raw: Record<string, unknown>,
): PushNotificationData {
  const type = raw.type as PushNotificationData['type'] | undefined;
  const route = typeof raw.route === 'string' ? raw.route : undefined;

  let params: PushNotificationData['params'] = {};
  if (raw.params != null && typeof raw.params === 'object') {
    params = parsePushParams(raw.params as Record<string, unknown>);
  } else {
    params = parsePushParams(raw);
  }

  return {type, route, params};
}

function parsePushParams(
  raw: Record<string, unknown>,
): PushNotificationData['params'] {
  const params: PushNotificationData['params'] = {};
  if (raw.debateId != null && raw.debateId !== '') {
    params.debateId = Number(raw.debateId);
  }
  if (raw.matchId != null && raw.matchId !== '') {
    params.matchId =
      typeof raw.matchId === 'number' ? raw.matchId : String(raw.matchId);
  }
  if (typeof raw.url === 'string') {
    params.url = raw.url;
  }
  if (typeof raw.shortVideoId === 'string') {
    params.shortVideoId = raw.shortVideoId;
  }
  return params;
}

export function parseMatchId(
  value: number | string | undefined,
): number | null {
  if (value == null || value === '') {
    return null;
  }
  const n = typeof value === 'number' ? value : parseInt(String(value), 10);
  return Number.isFinite(n) && n > 0 ? n : null;
}

/** Minimal match shell when fixture metadata is not yet loaded. */
export function buildPlaceholderMatchFromId(
  matchId: number,
  opts?: {statusShort?: string; date?: string},
): Match {
  const statusShort = opts?.statusShort ?? 'NS';
  const statusLong =
    statusShort === 'FT' ? 'Full Time' : statusShort === 'NS' ? 'Not Started' : statusShort;
  return {
    fixture: {
      id: matchId,
      date: opts?.date ?? new Date().toISOString(),
      status: {long: statusLong, short: statusShort, elapsed: 0},
    },
    league: {id: 0, name: '', logo: '', season: new Date().getFullYear()},
    teams: {
      home: {name: 'Home', logo: '', winner: null},
      away: {name: 'Away', logo: '', winner: null},
    },
    goals: {home: null, away: null},
  };
}

/** Build a Match from debate API fields (teams/score when present). */
export function buildMatchFromDebate(debate: DebateResponse): Match {
  const matchId = parseMatchId(debate.match_id) ?? 0;
  const home = debate.teams?.home;
  const away = debate.teams?.away;
  return {
    fixture: {
      id: matchId,
      date: debate.created_at ?? new Date().toISOString(),
      status: {long: 'Full Time', short: 'FT', elapsed: 0},
    },
    league: {id: 0, name: '', logo: '', season: new Date().getFullYear()},
    teams: {
      home: {
        name: home?.name ?? 'Home',
        logo: home?.logo ?? '',
        winner: null,
      },
      away: {
        name: away?.name ?? 'Away',
        logo: away?.logo ?? '',
        winner: null,
      },
    },
    goals: {
      home: home?.score ?? null,
      away: away?.score ?? null,
    },
  };
}

/** Maps notification `data` to a navigation target after optional prefetch. */
export function resolvePushNavigation(
  data: PushNotificationData,
  context?: PushPrefetchContext,
): PushNavigationTarget | null {
  const route = data.route ?? inferRouteFromType(data.type);
  const params = data.params ?? {};

  if (route === 'NewsWebView' || data.type === 'news') {
    const url = params.url;
    if (typeof url === 'string' && url.startsWith('http')) {
      return {kind: 'news', url};
    }
    return null;
  }

  if (route === 'SingleDebate' || data.type === 'debate') {
    if (context?.debate && context?.match) {
      return {
        kind: 'debate',
        match: context.match,
        debate: context.debate,
      };
    }
    return {kind: 'debates_tab'};
  }

  if (route === 'MatchDetails' || data.type === 'match') {
    if (context?.match) {
      return {kind: 'match', match: context.match};
    }
    return {kind: 'home_tab'};
  }

  return null;
}

function inferRouteFromType(
  type?: PushNotificationData['type'],
): string | undefined {
  switch (type) {
    case 'news':
      return 'NewsWebView';
    case 'debate':
      return 'SingleDebate';
    case 'match':
      return 'MatchDetails';
    default:
      return undefined;
  }
}
