import type {QueryClient} from '@tanstack/react-query';
import {WORLD_CUP_ONLY_MODE} from '../config/featureFlags';
import {
  DEFAULT_LEAGUE,
  WORLD_CUP_LEAGUE,
  seasonParamForMatchSearch,
  type League,
} from '../constants/leagues';
import {matchesForLocalDateQueryKey} from '../queries/keys';
import {resolveHomeScreenDefaultLeague} from '../services/matchesDefaultLeague';
import {fetchDebateById} from '../services/debate';
import {fetchMatchesForLocalDate} from '../services/futbol';
import {
  fetchMatchShorts,
  MATCH_SHORTS_STALE_MS,
  matchShortsQueryKey,
} from '../services/matchShortsApi';
import type {Match} from '../types/match';
import type {DebateResponse} from '../types/debate';
import {
  buildMatchFromDebate,
  buildPlaceholderMatchFromId,
  normalizePushNotificationData,
  parseMatchId,
  type PushNotificationData,
  type PushPrefetchContext,
} from './pushLinking';

function startOfLocalDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function findMatchInQueryCache(
  queryClient: QueryClient,
  matchId: number,
): Match | null {
  const queries = queryClient.getQueryCache().findAll({
    queryKey: ['matches', 'localDate'],
  });
  for (const query of queries) {
    const matches = query.state.data as Match[] | undefined;
    const found = matches?.find(m => m.fixture.id === matchId);
    if (found) {
      return found;
    }
  }
  return null;
}

async function resolvePrefetchLeague(today: Date): Promise<League> {
  if (WORLD_CUP_ONLY_MODE) {
    return WORLD_CUP_LEAGUE;
  }
  try {
    return await resolveHomeScreenDefaultLeague(today);
  } catch {
    return DEFAULT_LEAGUE;
  }
}

/** Load fixture metadata from recent match-list caches or API. */
export async function resolveMatchForPush(
  queryClient: QueryClient,
  matchId: number,
  opts?: {statusShort?: string},
): Promise<Match> {
  const cached = findMatchInQueryCache(queryClient, matchId);
  if (cached) {
    return cached;
  }

  const today = startOfLocalDay(new Date());
  const league = await resolvePrefetchLeague(today);
  const season = seasonParamForMatchSearch(league, today);

  for (const offset of [-2, -1, 0, 1, 2]) {
    const date = new Date(today);
    date.setDate(date.getDate() + offset);
    const key = matchesForLocalDateQueryKey(date, league.id);
    let matches = queryClient.getQueryData<Match[] | null>(key);
    if (!matches) {
      try {
        matches = await fetchMatchesForLocalDate(date, league.id, season);
        queryClient.setQueryData(key, matches);
      } catch {
        continue;
      }
    }
    const found = matches?.find(m => m.fixture.id === matchId);
    if (found) {
      return found;
    }
  }

  return buildPlaceholderMatchFromId(matchId, {
    statusShort: opts?.statusShort ?? 'FT',
  });
}

async function prefetchMatchShorts(
  queryClient: QueryClient,
  matchId: number,
): Promise<void> {
  await queryClient.prefetchQuery({
    queryKey: matchShortsQueryKey(matchId),
    queryFn: () => fetchMatchShorts(matchId),
    staleTime: MATCH_SHORTS_STALE_MS,
    gcTime: MATCH_SHORTS_STALE_MS,
  });
}

/** Warm debate/match data before navigating from a push tap. */
export async function prefetchPushContext(
  data: PushNotificationData,
  opts: {token?: string | null; queryClient: QueryClient},
): Promise<PushPrefetchContext> {
  const params = data.params ?? {};
  const route = data.route ?? data.type;
  const context: PushPrefetchContext = {};

  if (route === 'SingleDebate' || data.type === 'debate') {
    const debateId = params.debateId;
    if (debateId != null && Number.isFinite(debateId)) {
      const debate = await fetchDebateById(debateId, opts.token);
      if (debate) {
        context.debate = debate;
        const matchId =
          parseMatchId(params.matchId) ?? parseMatchId(debate.match_id);
        if (matchId != null) {
          context.match = await resolveMatchForPush(opts.queryClient, matchId, {
            statusShort: 'FT',
          });
        } else {
          context.match = buildMatchFromDebate(debate);
        }
      }
    }
    return context;
  }

  if (route === 'MatchDetails' || data.type === 'match') {
    const matchId = parseMatchId(params.matchId);
    if (matchId != null) {
      context.match = await resolveMatchForPush(opts.queryClient, matchId, {
        statusShort: 'FT',
      });
      void prefetchMatchShorts(opts.queryClient, matchId);
    }
    return context;
  }

  return context;
}

/** Parse raw Expo data, prefetch, and resolve a navigation target. */
export async function resolvePushNavigationFromRaw(
  raw: Record<string, unknown>,
  opts: {token?: string | null; queryClient: QueryClient},
) {
  const data = normalizePushNotificationData(raw);
  const context = await prefetchPushContext(data, opts);
  return resolvePushNavigation(data, context);
}

// Re-export for callers that already import from pushLinking.
export {resolvePushNavigation} from './pushLinking';
