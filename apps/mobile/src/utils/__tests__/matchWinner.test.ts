import type {Match} from '../types/match';
import {
  getMatchWinnerSide,
  isAwayMatchWinner,
  isHomeMatchWinner,
} from '../matchWinner';

function sampleMatch(overrides: Partial<Match> & {status?: string}): Match {
  const {status = 'FT', ...rest} = overrides;
  return {
    fixture: {
      id: 1,
      date: '2026-07-03T18:00:00Z',
      status: {long: 'Match Finished', short: status, elapsed: 90},
    },
    league: {id: 1, name: 'WC', logo: '', season: 2026},
    teams: {
      home: {name: 'Australia', logo: '', winner: null},
      away: {name: 'Egypt', logo: '', winner: null},
    },
    goals: {home: 1, away: 1},
    ...rest,
  };
}

describe('matchWinner', () => {
  it('uses teams.winner for penalty shootouts when goals are level', () => {
    const match = sampleMatch({
      status: 'PEN',
      teams: {
        home: {name: 'Australia', logo: '', winner: true},
        away: {name: 'Egypt', logo: '', winner: false},
      },
      goals: {home: 1, away: 1},
    });
    expect(getMatchWinnerSide(match)).toBe('home');
    expect(isHomeMatchWinner(match)).toBe(true);
    expect(isAwayMatchWinner(match)).toBe(false);
  });

  it('uses goal difference for regular full-time wins', () => {
    const match = sampleMatch({
      status: 'FT',
      teams: {
        home: {name: 'Colombia', logo: '', winner: true},
        away: {name: 'Ghana', logo: '', winner: false},
      },
      goals: {home: 1, away: 0},
    });
    expect(getMatchWinnerSide(match)).toBe('home');
  });

  it('returns null for draws with no winner flag', () => {
    const match = sampleMatch({
      status: 'FT',
      teams: {
        home: {name: 'A', logo: '', winner: false},
        away: {name: 'B', logo: '', winner: false},
      },
      goals: {home: 0, away: 0},
    });
    expect(getMatchWinnerSide(match)).toBe(null);
  });
});
