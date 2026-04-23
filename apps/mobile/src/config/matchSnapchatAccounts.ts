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
    teamNameKey: 'fc bayern munich',
    snapchatUsername: 'fcbayern',
  },
  {
    teamNameKey: 'atletico madrid',
    snapchatUsername: 'atleti',
  },
  {
    teamNameKey: 'real madrid',
    snapchatUsername: 'realmadrid',
  },
  {
    teamNameKey: 'manchester city',
    snapchatUsername: 'mancityofficial',
  },
  {
    teamNameKey: 'chelsea',
    snapchatUsername: 'chelseafc',
  },
  {
    teamNameKey: 'liverpool',
    snapchatUsername: 'lfc',
  },
  {
    teamNameKey: 'manchester united',
    snapchatUsername: 'manutd',
  },
  {
    teamNameKey: 'tottenham hotspur',
    snapchatUsername: 'spursofficial',
  },
  {
    teamNameKey: 'arsenal',
    snapchatUsername: 'arsenal',
  },
  {
    teamNameKey: 'west ham united',
    snapchatUsername: 'whufcofficial',
  },
  {
    teamNameKey: 'aston villa',
    snapchatUsername: 'avfc',
  },
  {
    teamNameKey: 'everton',
    snapchatUsername: 'everton',
  },
  {
    teamNameKey: 'leicester city',
    snapchatUsername: 'lcfcofficial',
  },
  {
    teamNameKey: 'newcastle united',
    snapchatUsername: 'nufc',
  },
  {
    teamNameKey: 'southampton',
    snapchatUsername: 'southamptonfc',
  },
  {
    teamNameKey: 'west bromwich albion',
    snapchatUsername: 'wba',
  },
  {
    teamNameKey: 'wolverhampton wanderers',
    snapchatUsername: 'wolves',
  },
  {
    teamNameKey: 'crystal palace',
    snapchatUsername: 'cpfc',
  },
  {
    teamNameKey: 'leeds united',
    snapchatUsername: 'lufc',
  },
  {
    teamNameKey: 'norwich city',
    snapchatUsername: 'norwichcityfc',
  },
  {
    teamNameKey: 'juventus',
    snapchatUsername: 'juventus',
  },
  {
    teamNameKey: 'inter milan',
    snapchatUsername: 'interoffcial',
  },
  {
    teamNameKey: 'as roma',
    snapchatUsername: 'officialasroma',
  },
];

export function snapchatUsernameForTeamName(teamName: string): string | null {
  const key = normalizeMatchTeamNameForLookup(teamName);
  const row = MATCH_TEAM_SNAPCHAT_USERNAMES.find(r => r.teamNameKey === key);
  return row?.snapchatUsername ?? null;
}
