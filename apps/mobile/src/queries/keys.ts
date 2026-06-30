/** Stable local calendar day (device timezone) for match tab cache keys. */
export function localCalendarDayKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

export const newsFootballQueryKey = ['news', 'football'] as const;

export function matchesForLocalDateQueryKey(localDate: Date, leagueId: number) {
  return ['matches', 'localDate', localCalendarDayKey(localDate), leagueId] as const;
}

export function mainDebatesFeedQueryKey(
  token: string | null | undefined,
  userId: number | undefined,
) {
  return token
    ? (['mainDebatesFeed', userId ?? 'auth'] as const)
    : (['mainDebatesFeed', 'guest'] as const);
}

export const mainDebatesFeedGuestQueryKey = ['mainDebatesFeed', 'guest'] as const;
