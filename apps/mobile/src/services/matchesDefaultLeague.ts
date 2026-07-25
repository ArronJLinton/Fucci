import {LEAGUES, MLS_LEAGUE_ID} from '../constants/leagues';
import type {League} from '../constants/leagues';

function requireLeague(id: number): League {
  const league = LEAGUES.find(l => l.id === id);
  if (league == null) {
    throw new Error(`matchesDefaultLeague: expected league id ${id} in LEAGUES`);
  }
  return league;
}

const MLS_LEAGUE = requireLeague(MLS_LEAGUE_ID);

function calendarDayKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(
    d.getDate(),
  ).padStart(2, '0')}`;
}

/** One resolved default per calendar day per app session (avoids repeat probes on Home remount). */
let sessionDefaultLeagueByDay: {dayKey: string; league: League} | null = null;

/** Sat / Sun / Mon → Premier League preferred (re-enable with PL-season default logic). */
export function isPremierLeaguePreferredDay(d: Date): boolean {
  const day = d.getDay();
  return day === 0 || day === 1 || day === 6;
}

/**
 * Home Matches strip default.
 * Temporary: always MLS until the Premier League season starts, then restore
 * Sat–Mon → Premier League / Tue–Fri → UCL if fixtures today else Premier.
 */
export async function resolveHomeScreenDefaultLeague(
  todayLocal: Date,
): Promise<League> {
  const today = new Date(
    todayLocal.getFullYear(),
    todayLocal.getMonth(),
    todayLocal.getDate(),
  );

  const dayKey = calendarDayKey(today);
  if (sessionDefaultLeagueByDay?.dayKey === dayKey) {
    return sessionDefaultLeagueByDay.league;
  }

  sessionDefaultLeagueByDay = {dayKey, league: MLS_LEAGUE};
  return MLS_LEAGUE;
}
