import type {Match} from '../types/match';

/** Normalize API-Football short status codes (aligned with backend cache mapping). */
export function normalizeMatchStatus(status: string): string {
  return status.trim().toUpperCase();
}

const LIVE_MATCH_STATUSES = new Set([
  '1H',
  '2H',
  'HT',
  'ET',
  'P',
  'BT',
  'LIVE',
  'SUSP',
  'INT',
  'IN_PLAY',
]);

const SCHEDULED_MATCH_STATUSES = new Set(['NS', 'TBD', 'SCHEDULED']);

const FINISHED_MATCH_STATUSES = new Set([
  'FT',
  'AET',
  'PEN',
  'FT_PEN',
  'AET_PEN',
  'AWD',
  'WO',
  'CANC',
  'ABD',
  'FINISHED',
]);

/** Fixtures that actually completed (excludes walkover/cancelled for news/debate context). */
const PLAYED_MATCH_STATUSES = new Set([
  'FT',
  'AET',
  'PEN',
  'FT_PEN',
  'AET_PEN',
]);

export function isLiveMatchStatus(status: string): boolean {
  return LIVE_MATCH_STATUSES.has(normalizeMatchStatus(status));
}

export function isScheduledMatchStatus(status: string): boolean {
  return SCHEDULED_MATCH_STATUSES.has(normalizeMatchStatus(status));
}

export function isFinishedMatchStatus(status: string): boolean {
  return FINISHED_MATCH_STATUSES.has(normalizeMatchStatus(status));
}

export function isPlayedMatchStatus(status: string): boolean {
  return PLAYED_MATCH_STATUSES.has(normalizeMatchStatus(status));
}

export function hasLiveMatchInList(matches: Match[]): boolean {
  return matches.some(m => isLiveMatchStatus(m.fixture.status.short));
}

export function resolveRefreshedMatch(
  initialMatch: Match,
  matches: Match[] | undefined,
): Match {
  return (
    matches?.find(
      candidate => candidate.fixture.id === initialMatch.fixture.id,
    ) ?? initialMatch
  );
}

export function shouldPollLiveMatch(
  initialMatch: Match,
  matches: Match[] | undefined,
): boolean {
  return isLiveMatchStatus(
    resolveRefreshedMatch(initialMatch, matches).fixture.status.short,
  );
}
