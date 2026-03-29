export interface League {
  id: number;
  name: string;
  logo?: string;
}

export const LEAGUES: League[] = [
  {id: 39, name: 'Premier League'},
  {id: 140, name: 'La Liga'},
  {id: 135, name: 'Serie A'},
  {id: 78, name: 'Bundesliga'},
  {id: 61, name: 'Ligue 1'},
  {id: 2, name: 'UEFA Champions League'},
  {id: 0, name: 'International Competitions'},
];

export const DEFAULT_LEAGUE = LEAGUES[0]; // Premier League
