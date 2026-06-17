/// <reference types="jest" />

// Lock the test process to Eastern Time so we exercise the actual reported
// scenario (9 PM ET kickoff = 01:00 UTC next day).
process.env.TZ = 'America/New_York';

import * as api from '../api';
import {fetchMatchesForLocalDate} from '../futbol';

// Minimal fixture shape; tests only inspect the fields the production code
// uses (fixture.id, fixture.date). League/teams/status are populated to
// satisfy the Match interface at runtime.
type FakeMatch = {
  fixture: {id: number; date: string; status: string};
  teams: {
    home: {id: number; name: string; logo: string};
    away: {id: number; name: string; logo: string};
  };
  league: {id: number; name: string; season: number};
};

function fakeFixture(
  id: number,
  isoUtc: string,
  home = 'Home',
  away = 'Away',
): FakeMatch {
  return {
    fixture: {id, date: isoUtc, status: 'NS'},
    teams: {
      home: {id: id * 10, name: home, logo: ''},
      away: {id: id * 10 + 1, name: away, logo: ''},
    },
    league: {id: 1, name: 'FIFA World Cup', season: 2026},
  };
}

describe('fetchMatchesForLocalDate (TZ=America/New_York)', () => {
  let makeApiRequest: jest.SpyInstance;

  beforeEach(() => {
    makeApiRequest = jest.spyOn(api, 'makeApiRequest');
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('fetches both overlapping UTC dates and re-buckets by local date', async () => {
    // Setup: viewer in ET asks for "Tue Jun 16 local" tab.
    // ET Tue Jun 16 spans UTC Jun 16 04:00 → UTC Jun 17 03:59, so the helper
    // must hit the API twice (once per UTC date) and then filter.
    const argentinaAlgeria = fakeFixture(
      1001,
      '2026-06-17T01:00:00+00:00',
      'Argentina',
      'Algeria',
    );
    const earlierGame = fakeFixture(
      1002,
      '2026-06-16T18:00:00+00:00',
      'France',
      'Iran',
    );
    const austriaJordan = fakeFixture(
      1003,
      '2026-06-17T04:00:00+00:00',
      'Austria',
      'Jordan',
    );

    makeApiRequest.mockImplementation(async (endpoint: string) => {
      if (endpoint.includes('date=2026-06-16')) {
        return {response: [earlierGame]};
      }
      if (endpoint.includes('date=2026-06-17')) {
        return {response: [argentinaAlgeria, austriaJordan]};
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    });

    const localTue = new Date(2026, 5, 16); // Tue Jun 16 ET
    const result = await fetchMatchesForLocalDate(localTue, 1);

    // Two HTTP calls, one per overlapping UTC date.
    expect(makeApiRequest).toHaveBeenCalledTimes(2);
    expect(makeApiRequest.mock.calls[0][0]).toContain('date=2026-06-16');
    expect(makeApiRequest.mock.calls[1][0]).toContain('date=2026-06-17');

    // Tue ET should contain France-Iran (afternoon ET) AND Argentina-Algeria
    // (9 PM ET = 01:00 UTC Wed) -- but NOT Austria-Jordan (midnight ET Wed).
    const ids = (result ?? []).map(m => m.fixture.id).sort();
    expect(ids).toEqual([1001, 1002]);
  });

  it('places Argentina vs Algeria on Tue (the bug fix)', async () => {
    const argentinaAlgeria = fakeFixture(
      1001,
      '2026-06-17T01:00:00+00:00',
      'Argentina',
      'Algeria',
    );
    const austriaJordan = fakeFixture(
      1003,
      '2026-06-17T04:00:00+00:00',
      'Austria',
      'Jordan',
    );

    makeApiRequest.mockImplementation(async (endpoint: string) => {
      if (endpoint.includes('date=2026-06-17')) {
        return {response: [argentinaAlgeria, austriaJordan]};
      }
      return {response: []};
    });

    const localTue = new Date(2026, 5, 16);
    const resultTue = await fetchMatchesForLocalDate(localTue, 1);
    const tueIds = (resultTue ?? []).map(m => m.fixture.id);
    expect(tueIds).toContain(1001); // Argentina-Algeria
    expect(tueIds).not.toContain(1003); // Austria-Jordan stays on Wed
  });

  it('places Austria vs Jordan on Wed and Argentina vs Algeria not on Wed', async () => {
    const argentinaAlgeria = fakeFixture(
      1001,
      '2026-06-17T01:00:00+00:00',
      'Argentina',
      'Algeria',
    );
    const austriaJordan = fakeFixture(
      1003,
      '2026-06-17T04:00:00+00:00',
      'Austria',
      'Jordan',
    );

    makeApiRequest.mockImplementation(async (endpoint: string) => {
      if (endpoint.includes('date=2026-06-17')) {
        return {response: [argentinaAlgeria, austriaJordan]};
      }
      return {response: []};
    });

    const localWed = new Date(2026, 5, 17);
    const resultWed = await fetchMatchesForLocalDate(localWed, 1);
    const wedIds = (resultWed ?? []).map(m => m.fixture.id);
    expect(wedIds).toContain(1003); // Austria-Jordan
    expect(wedIds).not.toContain(1001); // Argentina-Algeria is a Tue ET game
  });

  it('dedupes fixtures that appear in both UTC responses', async () => {
    const fxt = fakeFixture(2001, '2026-06-17T01:00:00+00:00', 'A', 'B');
    // Pathologically return the same fixture from both UTC date queries.
    makeApiRequest.mockResolvedValue({response: [fxt]});

    const localTue = new Date(2026, 5, 16);
    const result = await fetchMatchesForLocalDate(localTue, 1);

    expect(result).toHaveLength(1);
    expect(result?.[0].fixture.id).toBe(2001);
  });

  it('sorts results by kickoff ascending across UTC buckets', async () => {
    const lateUtc = fakeFixture(1, '2026-06-17T01:00:00+00:00');
    const earlyUtc = fakeFixture(2, '2026-06-16T18:00:00+00:00');

    makeApiRequest.mockImplementation(async (endpoint: string) => {
      if (endpoint.includes('date=2026-06-16'))
        return {response: [earlyUtc]};
      if (endpoint.includes('date=2026-06-17'))
        return {response: [lateUtc]};
      return {response: []};
    });

    const result = await fetchMatchesForLocalDate(new Date(2026, 5, 16), 1);
    expect((result ?? []).map(m => m.fixture.id)).toEqual([2, 1]);
  });

  it('forwards league_id and season query params on every UTC fetch', async () => {
    makeApiRequest.mockResolvedValue({response: []});

    await fetchMatchesForLocalDate(new Date(2026, 5, 16), 1, 2026);

    expect(makeApiRequest).toHaveBeenCalledTimes(2);
    for (const call of makeApiRequest.mock.calls) {
      const endpoint = call[0] as string;
      expect(endpoint).toContain('league_id=1');
      expect(endpoint).toContain('season=2026');
    }
  });

  it('returns [] (not null) when both UTC days are empty', async () => {
    makeApiRequest.mockResolvedValue({response: []});
    const result = await fetchMatchesForLocalDate(new Date(2026, 5, 16), 1);
    expect(result).toEqual([]);
  });

  it('tolerates a partial upstream failure (one UTC fetch fails)', async () => {
    const ok = fakeFixture(99, '2026-06-16T18:00:00+00:00');
    let call = 0;
    makeApiRequest.mockImplementation(async (endpoint: string) => {
      call += 1;
      if (endpoint.includes('date=2026-06-17')) {
        // The inner per-UTC helper catches errors and returns null; verify
        // the outer caller still returns the successful bucket's fixtures.
        throw new Error('upstream 503');
      }
      return {response: [ok]};
    });

    const result = await fetchMatchesForLocalDate(new Date(2026, 5, 16), 1);
    expect(call).toBe(2);
    expect((result ?? []).map(m => m.fixture.id)).toEqual([99]);
  });
});
