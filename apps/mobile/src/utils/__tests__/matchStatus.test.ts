import type {Match} from '../../types/match';
import {getMatchRefetchInterval} from '../matchStatus';

function match(status: string, date: string): Match {
  return {
    fixture: {
      id: 1,
      date,
      status: {long: status, short: status, elapsed: 0},
    },
    league: {id: 1, name: 'World Cup', logo: '', season: 2026},
    teams: {
      home: {name: 'Home', logo: '', winner: null},
      away: {name: 'Away', logo: '', winner: null},
    },
    goals: {home: null, away: null},
  };
}

describe('getMatchRefetchInterval', () => {
  const now = Date.parse('2026-07-16T15:00:00Z');
  const liveInterval = 75_000;

  it('polls at the live interval when a fixture is live', () => {
    expect(
      getMatchRefetchInterval(
        [match('1H', '2026-07-16T14:30:00Z')],
        liveInterval,
        now,
      ),
    ).toBe(liveInterval);
  });

  it('refetches just after the nearest confirmed kickoff', () => {
    expect(
      getMatchRefetchInterval(
        [
          match('NS', '2026-07-16T16:00:00Z'),
          match('NS', '2026-07-16T15:10:00Z'),
        ],
        liveInterval,
        now,
      ),
    ).toBe(10 * 60_000 + 5_000);
  });

  it('keeps polling while an overdue fixture is still not started', () => {
    expect(
      getMatchRefetchInterval(
        [match('NS', '2026-07-16T14:59:00Z')],
        liveInterval,
        now,
      ),
    ).toBe(liveInterval);
  });

  it('does not poll finished or unconfirmed TBD fixtures', () => {
    expect(
      getMatchRefetchInterval(
        [
          match('FT', '2026-07-16T13:00:00Z'),
          match('TBD', '2026-07-16T14:00:00Z'),
        ],
        liveInterval,
        now,
      ),
    ).toBe(false);
  });
});
