import type {QueryClient} from '@tanstack/react-query';
import {matchesForLocalDateQueryKey} from '../../queries/keys';
import {fetchDebateById} from '../../services/debate';
import {fetchMatchesForLocalDate} from '../../services/futbol';
import {fetchMatchShorts} from '../../services/matchShortsApi';
import type {Match} from '../../types/match';
import {
  prefetchPushContext,
  resolveMatchForPush,
} from '../prefetchPushContext';

jest.mock('../../services/debate', () => ({
  fetchDebateById: jest.fn(),
}));

jest.mock('../../services/futbol', () => ({
  fetchMatchesForLocalDate: jest.fn(),
}));

jest.mock('../../services/matchShortsApi', () => ({
  fetchMatchShorts: jest.fn(),
  matchShortsQueryKey: (id: number) => ['matchShorts', id],
  MATCH_SHORTS_STALE_MS: 60_000,
}));

jest.mock('../../services/matchesDefaultLeague', () => ({
  resolveHomeScreenDefaultLeague: jest.fn().mockResolvedValue({id: 1, name: 'WC'}),
}));

const mockFetchDebateById = fetchDebateById as jest.MockedFunction<
  typeof fetchDebateById
>;
const mockFetchMatches = fetchMatchesForLocalDate as jest.MockedFunction<
  typeof fetchMatchesForLocalDate
>;
const mockFetchMatchShorts = fetchMatchShorts as jest.MockedFunction<
  typeof fetchMatchShorts
>;
type FetchMatchesForLocalDateResult = Awaited<
  ReturnType<typeof fetchMatchesForLocalDate>
>;

function createQueryClient(): QueryClient {
  const {QueryClient} = require('@tanstack/react-query');
  return new QueryClient({
    defaultOptions: {queries: {retry: false}},
  });
}

function sampleMatch(id: number): Match {
  return {
    fixture: {
      id,
      date: '2026-06-29T18:00:00Z',
      status: {long: 'Match Finished', short: 'FT', elapsed: 90},
    },
    league: {id: 1, name: 'WC', logo: '', season: 2026},
    teams: {
      home: {name: 'Home', logo: '', winner: true},
      away: {name: 'Away', logo: '', winner: false},
    },
    goals: {home: 1, away: 0},
  };
}

describe('prefetchPushContext', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockFetchMatchShorts.mockResolvedValue({
      match_id: '0',
      teams: {
        home: {
          lookup_key: 'home',
          has_shorts: false,
          shorts: [],
          has_user_stories: false,
          user_stories: [],
        },
        away: {
          lookup_key: 'away',
          has_shorts: false,
          shorts: [],
          has_user_stories: false,
          user_stories: [],
        },
      },
    });
  });

  it('prefetches debate and match for debate pushes', async () => {
    mockFetchDebateById.mockResolvedValue({
      id: 42,
      match_id: '99',
      headline: 'Test',
      description: '',
      cards: [],
      teams: {
        home: {name: 'A'},
        away: {name: 'B'},
      },
    });
    const matches: FetchMatchesForLocalDateResult = [sampleMatch(99)];
    mockFetchMatches.mockResolvedValue(matches);

    const queryClient = createQueryClient();
    const context = await prefetchPushContext(
      {type: 'debate', params: {debateId: 42, matchId: '99'}},
      {queryClient, token: 'tok'},
    );

    expect(mockFetchDebateById).toHaveBeenCalledWith(42, 'tok');
    expect(context.debate?.id).toBe(42);
    expect(context.match?.fixture.id).toBe(99);
    expect(context.match?.teams.home.name).toBe('Home');
  });

  it('prefetches match and shorts for match pushes', async () => {
    const matches: FetchMatchesForLocalDateResult = [sampleMatch(7)];
    mockFetchMatches.mockResolvedValue(matches);

    const queryClient = createQueryClient();
    const context = await prefetchPushContext(
      {type: 'match', params: {matchId: 7, shortVideoId: 'abc123'}},
      {queryClient},
    );

    expect(context.match?.fixture.id).toBe(7);
    expect(mockFetchMatchShorts).toHaveBeenCalledWith(7);
  });
});

describe('resolveMatchForPush', () => {
  it('returns cached match when present in React Query', async () => {
    const queryClient = createQueryClient();
    const today = new Date(2026, 5, 30);
    const cached = sampleMatch(5);
    queryClient.setQueryData(matchesForLocalDateQueryKey(today, 1), [cached]);

    const match = await resolveMatchForPush(queryClient, 5);
    expect(match).toEqual(cached);
  });
});
