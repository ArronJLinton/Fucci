import type {QueryClient} from '@tanstack/react-query';
import {matchesForLocalDateQueryKey} from '../../queries/keys';
import {fetchDebateById} from '../../services/debate';
import {fetchMatchesForLocalDate} from '../../services/futbol';
import {fetchMatchShorts} from '../../services/matchShortsApi';
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

function createQueryClient(): QueryClient {
  const {QueryClient} = require('@tanstack/react-query');
  return new QueryClient({
    defaultOptions: {queries: {retry: false}},
  });
}

describe('prefetchPushContext', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockFetchMatchShorts.mockResolvedValue({teams: {home: null, away: null}});
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
    mockFetchMatches.mockResolvedValue([
      {
        fixture: {id: 99, date: '2026-06-29T18:00:00Z', status: {long: 'FT', short: 'FT', elapsed: 0}},
        league: {id: 1, name: 'WC', logo: '', season: 2026},
        teams: {
          home: {name: 'A', logo: '', winner: true},
          away: {name: 'B', logo: '', winner: false},
        },
        goals: {home: 1, away: 0},
      },
    ]);

    const queryClient = createQueryClient();
    const context = await prefetchPushContext(
      {type: 'debate', params: {debateId: 42, matchId: '99'}},
      {queryClient, token: 'tok'},
    );

    expect(mockFetchDebateById).toHaveBeenCalledWith(42, 'tok');
    expect(context.debate?.id).toBe(42);
    expect(context.match?.fixture.id).toBe(99);
    expect(context.match?.teams.home.name).toBe('A');
  });

  it('prefetches match and shorts for match pushes', async () => {
    mockFetchMatches.mockResolvedValue([
      {
        fixture: {id: 7, date: '2026-06-29T18:00:00Z', status: {long: 'FT', short: 'FT', elapsed: 0}},
        league: {id: 1, name: 'WC', logo: '', season: 2026},
        teams: {
          home: {name: 'X', logo: '', winner: null},
          away: {name: 'Y', logo: '', winner: null},
        },
        goals: {home: 0, away: 0},
      },
    ]);

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
    const cached = {
      fixture: {id: 5, date: '2026-06-29T18:00:00Z', status: {long: 'FT', short: 'FT', elapsed: 0}},
      league: {id: 1, name: 'WC', logo: '', season: 2026},
      teams: {
        home: {name: 'H', logo: '', winner: null},
        away: {name: 'A', logo: '', winner: null},
      },
      goals: {home: null, away: null},
    };
    queryClient.setQueryData(matchesForLocalDateQueryKey(today, 1), [cached]);

    const match = await resolveMatchForPush(queryClient, 5);
    expect(match).toEqual(cached);
  });
});
