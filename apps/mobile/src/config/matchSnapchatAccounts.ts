/**
 * Map normalized `Match.teams.*.name` to RapidAPI Snapchat `username` query values.
 * Add one row per club; `teamNameKey` must match `normalizeMatchTeamNameForLookup(name)`.
 */
export function normalizeMatchTeamNameForLookup(name: string): string {
  return name
    .normalize('NFD')
    .replace(/\p{M}/gu, '')
    .toLowerCase()
    .replace(/[^a-z0-9\s]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

export const MATCH_TEAM_SNAPCHAT_USERNAMES: ReadonlyArray<{
  teamNameKey: string;
  snapchatUsername: string;
}> = [
  {teamNameKey: 'paris saint germain', snapchatUsername: 'psg'},
  /** API-Football often uses "Barcelona" without the FC prefix */
  {teamNameKey: 'barcelona', snapchatUsername: 'fcbarcelona'},
  {teamNameKey: 'fc barcelona', snapchatUsername: 'fcbarcelona'},
  {
    teamNameKey: 'FC Bayern Munich',
    snapchatUsername: 'fcbayern',
  },
  {
    teamNameKey: 'atletico madrid',
    snapchatUsername: 'atleti',
  },
];

export function snapchatUsernameForTeamName(teamName: string): string | null {
  const key = normalizeMatchTeamNameForLookup(teamName);
  const row = MATCH_TEAM_SNAPCHAT_USERNAMES.find(r => r.teamNameKey === key);
  return row?.snapchatUsername ?? null;
}
