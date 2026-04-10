export interface League {
  id: number;
  name: string;
  /** League badge from API-Sports CDN (`/football/leagues/{id}.png`). */
  logo?: string;
}

/** API-Football league id for UEFA Champions League (`/fixtures` season = calendar year of match day). */
export const UCL_LEAGUE_ID = 2;

/**
 * Optional `season` query for GET /futbol/matches. UCL uses the calendar year of the
 * requested match date (e.g. April 2026 → 2026); other leagues are resolved on the server.
 */
export function seasonParamForMatchSearch(
  league: League,
  matchDate: Date,
): number | undefined {
  if (league.id === UCL_LEAGUE_ID) {
    return matchDate.getFullYear();
  }
  return undefined;
}

/** Base URL for league logos (matches API-Football / `media.api-sports.io` responses). */
const L = (leagueId: number) =>
  `https://media.api-sports.io/football/leagues/${leagueId}.png`;

export const LEAGUES: League[] = [
  {id: 39, name: 'Premier League', logo: L(39)},
  {id: 140, name: 'La Liga', logo: L(140)},
  {id: 135, name: 'Serie A', logo: L(135)},
  {id: 78, name: 'Bundesliga', logo: L(78)},
  {id: 61, name: 'Ligue 1', logo: L(61)},
  {id: UCL_LEAGUE_ID, name: 'UEFA Champions League', logo: L(UCL_LEAGUE_ID)},
  // No API league id 0; use World Cup badge as intl / multi-nation stand-in.
  {id: 0, name: 'International Competitions', logo: L(1)},
];

export const DEFAULT_LEAGUE = LEAGUES[0]; // Premier League

/** Short labels for horizontal league strips (PREMIER, LA LIGA, …). */
export function leagueStripLabel(name: string): string {
  const map: Record<string, string> = {
    'Premier League': 'PREMIER',
    'La Liga': 'LA LIGA',
    'Serie A': 'SERIE A',
    Bundesliga: 'BUNDES',
    'Ligue 1': 'LIGUE 1',
    'UEFA Champions League': 'UCL',
    'International Competitions': 'INTL',
  };
  return map[name] ?? name.split(' ')[0]?.toUpperCase() ?? name;
}
