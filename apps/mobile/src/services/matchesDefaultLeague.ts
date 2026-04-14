import {fetchMatches} from './futbol';
import {LEAGUES, UCL_LEAGUE_ID, seasonParamForMatchSearch} from '../constants/leagues';
import type {League} from '../constants/leagues';

const PREMIER_LEAGUE = LEAGUES.find(l => l.id === 39) as League;
const UCL_LEAGUE = LEAGUES.find(l => l.id === UCL_LEAGUE_ID) as League;

/** API-Football rejects fixture queries too far ahead; keep probes within this window. */
const MAX_DAYS_AHEAD_FOR_UCL_PROBE = 10;

/** Sat / Sun / Mon → Premier League is the default strip selection on Home. */
export function isPremierLeaguePreferredDay(d: Date): boolean {
  const day = d.getDay();
  return day === 0 || day === 1 || day === 6;
}

/**
 * Tue–Fri: default to UCL if any UCL fixture exists in a forward window (today through
 * today + MAX_DAYS_AHEAD_FOR_UCL_PROBE), else Premier League. Sat–Mon always Premier League.
 */
export async function resolveHomeScreenDefaultLeague(
  todayLocal: Date,
): Promise<League> {
  const today = new Date(
    todayLocal.getFullYear(),
    todayLocal.getMonth(),
    todayLocal.getDate(),
  );

  if (isPremierLeaguePreferredDay(today)) {
    return PREMIER_LEAGUE;
  }

  const dates: Date[] = [];
  for (let i = 0; i <= MAX_DAYS_AHEAD_FOR_UCL_PROBE; i++) {
    const d = new Date(today);
    d.setDate(today.getDate() + i);
    dates.push(d);
  }

  const flags = await Promise.all(
    dates.map(async d => {
      const rows = await fetchMatches(
        d,
        UCL_LEAGUE_ID,
        seasonParamForMatchSearch(UCL_LEAGUE, d),
      );
      return rows != null && rows.length > 0;
    }),
  );

  return flags.some(Boolean) ? UCL_LEAGUE : PREMIER_LEAGUE;
}
