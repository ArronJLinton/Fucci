import type {QueryClient} from '@tanstack/react-query';
import {WORLD_CUP_ONLY_MODE} from '../config/featureFlags';
import {
  DEFAULT_LEAGUE,
  WORLD_CUP_LEAGUE,
  seasonParamForMatchSearch,
  type League,
} from '../constants/leagues';
import {resolveHomeScreenDefaultLeague} from '../services/matchesDefaultLeague';
import {fetchFootballNews} from '../services/newsService';
import {fetchDebatesPublicFeed} from '../services/debate';
import {fetchMatchesForLocalDate} from '../services/futbol';
import {
  fetchMatchShorts,
  MATCH_SHORTS_STALE_MS,
  matchShortsQueryKey,
} from '../services/matchShortsApi';
import {
  fetchMediaShorts,
  mediaShortsQueryKey,
  MEDIA_SHORTS_STALE_MS,
} from '../services/mediaShortsApi';
import type {Match} from '../types/match';
import {
  mainDebatesFeedGuestQueryKey,
  matchesForLocalDateQueryKey,
  newsFootballQueryKey,
} from '../queries/keys';

const NEWS_STALE_MS = 5 * 60 * 1000;
const MATCHES_STALE_MS = 5 * 60 * 1000;
const DEBATES_STALE_MS = 2 * 60 * 1000;
/** Cap team Shorts prefetches so splash stays fast on heavy match days. */
const MAX_MATCH_SHORTS_PREFETCH = 12;

function startOfLocalDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
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

async function prefetchMatchTeamShorts(
  queryClient: QueryClient,
  matches: Match[] | null | undefined,
): Promise<void> {
  if (!matches?.length) {
    return;
  }
  const ids = matches
    .slice(0, MAX_MATCH_SHORTS_PREFETCH)
    .map(m => m.fixture.id)
    .filter(id => Number.isFinite(id));

  await Promise.allSettled(
    ids.map(matchId =>
      queryClient.prefetchQuery({
        queryKey: matchShortsQueryKey(matchId),
        queryFn: () => fetchMatchShorts(matchId),
        staleTime: MATCH_SHORTS_STALE_MS,
        gcTime: MATCH_SHORTS_STALE_MS,
      }),
    ),
  );
}

/**
 * Seeds React Query during splash so first tab visits reuse in-memory cache.
 * Also warms server Redis (news, matches, Shorts) for other users.
 */
export async function warmAppCache(queryClient: QueryClient): Promise<void> {
  const today = startOfLocalDay(new Date());
  const league = await resolvePrefetchLeague(today);
  const season = seasonParamForMatchSearch(league, today);

  const newsPrefetch = queryClient.prefetchQuery({
    queryKey: newsFootballQueryKey,
    queryFn: fetchFootballNews,
    staleTime: NEWS_STALE_MS,
    gcTime: 15 * 60 * 1000,
  });

  const mediaShortsPrefetch = queryClient.prefetchQuery({
    queryKey: mediaShortsQueryKey,
    queryFn: fetchMediaShorts,
    staleTime: MEDIA_SHORTS_STALE_MS,
    gcTime: MEDIA_SHORTS_STALE_MS,
  });

  const debatesPrefetch = queryClient.prefetchQuery({
    queryKey: mainDebatesFeedGuestQueryKey,
    queryFn: async () => {
      const pub = await fetchDebatesPublicFeed(30);
      return {kind: 'public' as const, debates: pub.debates};
    },
    staleTime: DEBATES_STALE_MS,
    gcTime: 10 * 60 * 1000,
  });

  const matchesKey = matchesForLocalDateQueryKey(today, league.id);
  const matchesPrefetch = queryClient
    .prefetchQuery({
      queryKey: matchesKey,
      queryFn: () => fetchMatchesForLocalDate(today, league.id, season),
      staleTime: MATCHES_STALE_MS,
      gcTime: 15 * 60 * 1000,
    })
    .then(() => {
      const matches = queryClient.getQueryData<Match[] | null>(matchesKey);
      return prefetchMatchTeamShorts(queryClient, matches ?? undefined);
    });

  const results = await Promise.allSettled([
    newsPrefetch,
    mediaShortsPrefetch,
    debatesPrefetch,
    matchesPrefetch,
  ]);

  for (const result of results) {
    if (result.status === 'rejected') {
      console.warn('[warmAppCache] prefetch failed:', result.reason);
    }
  }
}
