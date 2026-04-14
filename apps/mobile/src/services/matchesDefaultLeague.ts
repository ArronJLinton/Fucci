import {fetchMatches} from './futbol';
import {LEAGUES, UCL_LEAGUE_ID, seasonParamForMatchSearch} from '../constants/leagues';
import type {League} from '../constants/leagues';

const PREMIER_LEAGUE = LEAGUES.find(l => l.id === 39) as League;
const UCL_LEAGUE = LEAGUES.find(l => l.id === UCL_LEAGUE_ID) as League;

function calendarDayKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(
    d.getDate(),
  ).padStart(2, '0')}`;
}

/** One resolved default per calendar day per app session (avoids repeat probes on Home remount). */
let sessionDefaultLeagueByDay: {dayKey: string; league: League} | null = null;

/** Sat / Sun / Mon → Premier League is the default strip selection on Home. */
export function isPremierLeaguePreferredDay(d: Date): boolean {
  const day = d.getDay();
  return day === 0 || day === 1 || day === 6;
}

/**
 * Tue–Fri: default to UCL if there is at least one UCL fixture **today** (single fetch),
 * else Premier League. Sat–Mon always Premier League (no network).
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

  if (isPremierLeaguePreferredDay(today)) {
    sessionDefaultLeagueByDay = {dayKey, league: PREMIER_LEAGUE};
    return PREMIER_LEAGUE;
  }

  const rows = await fetchMatches(
    today,
    UCL_LEAGUE_ID,
    seasonParamForMatchSearch(UCL_LEAGUE, today),
  );
  const league =
    rows != null && rows.length > 0 ? UCL_LEAGUE : PREMIER_LEAGUE;
  sessionDefaultLeagueByDay = {dayKey, league};
  return league;
}
