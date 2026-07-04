import type {Match} from '../types/match';

export type MatchWinnerSide = 'home' | 'away' | null;

function teamWon(winner: boolean | null | undefined): boolean {
  return winner === true;
}

/**
 * Which side won the fixture. Uses API `teams.*.winner` (required for PEN when
 * regulation/extra-time goals are level), then falls back to goal difference.
 */
export function getMatchWinnerSide(match: Match): MatchWinnerSide {
  const homeW = teamWon(match.teams.home.winner);
  const awayW = teamWon(match.teams.away.winner);
  if (homeW && !awayW) {
    return 'home';
  }
  if (awayW && !homeW) {
    return 'away';
  }

  const h = match.goals.home;
  const a = match.goals.away;
  if (h != null && a != null && h !== a) {
    return h > a ? 'home' : 'away';
  }
  return null;
}

export function isHomeMatchWinner(match: Match): boolean {
  return getMatchWinnerSide(match) === 'home';
}

export function isAwayMatchWinner(match: Match): boolean {
  return getMatchWinnerSide(match) === 'away';
}
