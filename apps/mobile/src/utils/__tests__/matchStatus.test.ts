import type {Match} from '../../types/match';
import {
  resolveRefreshedMatch,
  shouldPollLiveMatch,
} from '../matchStatus';

function makeMatch(id: number, status: string, homeGoals: number): Match {
  return {
    fixture: {
      id,
      date: '2026-07-20T19:00:00Z',
      status: {long: status, short: status, elapsed: 45},
    },
    league: {
      id: 1,
      name: 'FIFA World Cup',
      logo: '',
      season: 2026,
    },
    teams: {
      home: {name: 'Home', logo: '', winner: null},
      away: {name: 'Away', logo: '', winner: null},
    },
    goals: {home: homeGoals, away: 0},
  };
}

describe('live match refresh', () => {
  it('uses refreshed fixture data when the match is present', () => {
    const initial = makeMatch(42, '1H', 0);
    const refreshed = makeMatch(42, '2H', 2);

    expect(resolveRefreshedMatch(initial, [refreshed])).toBe(refreshed);
  });

  it('keeps polling through an empty or unrelated refresh response', () => {
    const initial = makeMatch(42, '1H', 0);

    expect(shouldPollLiveMatch(initial, [])).toBe(true);
    expect(shouldPollLiveMatch(initial, [makeMatch(7, 'FT', 1)])).toBe(true);
  });

  it('stops polling after the refreshed fixture finishes', () => {
    const initial = makeMatch(42, '2H', 1);
    const finished = makeMatch(42, 'FT', 2);

    expect(shouldPollLiveMatch(initial, [finished])).toBe(false);
  });
});
