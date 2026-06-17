import {makeApiRequest} from './api';
import {
  formatLocalDate,
  isOnLocalDate,
  utcDateStringsForLocalDate,
} from '../utils/dateBuckets';

export interface Standing {
  rank: number;
  team: {
    id: number;
    name: string;
    logo: string;
  };
  points: number;
  all: {
    played: number;
    win: number;
    draw: number;
    lose: number;
    goals: {
      for: number;
      against: number;
    };
  };
  goalDifference: number;
}

export interface Match {
  fixture: {
    id: number;
    date: string;
    status: string;
  };
  teams: {
    home: {
      id: number;
      name: string;
      logo: string;
    };
    away: {
      id: number;
      name: string;
      logo: string;
    };
  };
  league: {
    id: number;
    name: string;
    season: number;
  };
}

export interface LineupData {
  home: {
    starters: any[];
    substitutes: any[];
  };
  away: {
    starters: any[];
    substitutes: any[];
  };
}

/**
 * Fetches fixtures for a single UTC calendar day. The `utcDate` string is
 * passed through verbatim to API-Football's `/fixtures?date=...` which
 * interprets it as UTC. Most callers should prefer fetchMatchesForLocalDate,
 * which handles the local-tz ↔ UTC bucketing correctly.
 */
const fetchMatchesByUtcDate = async (
  utcDate: string,
  leagueId?: number,
  season?: number,
): Promise<Match[] | null> => {
  try {
    let endpoint = `/futbol/matches?date=${utcDate}`;
    if (leagueId) {
      endpoint += `&league_id=${leagueId}`;
    }
    if (season !== undefined) {
      endpoint += `&season=${season}`;
    }
    const data = await makeApiRequest(endpoint, 'GET');
    return data.response ?? null;
  } catch (error) {
    console.error('Error fetching matches:', error);
    return null;
  }
};

/**
 * Backward-compatible single-date-param fetch (legacy): formats the query date
 * using the device's local calendar day and issues one request.
 *
 * For the user-facing YESTERDAY/TODAY/TOMORROW tabs use fetchMatchesForLocalDate
 * instead — it correctly handles local-day ↔ UTC bucketing.
 */
export const fetchMatches = async (
  date: Date,
  leagueId?: number,
  /** Rare override for `season`; normally omitted — server resolves per league + date. */
  season?: number,
): Promise<Match[] | null> => {
  return fetchMatchesByUtcDate(formatLocalDate(date), leagueId, season);
};

/**
 * Returns fixtures whose kickoff falls on the given local calendar date, in
 * the *device's* timezone. This is the function backing the
 * YESTERDAY/TODAY/TOMORROW tabs.
 *
 * Why this exists: API-Football groups fixtures by UTC date, so a 9 PM ET
 * Tuesday game (= 01:00 UTC Wednesday) gets returned for the Wednesday UTC
 * bucket — and would show up under the wrong tab for any user east or west
 * of UTC. We fix it by fetching every UTC date that overlaps the local day
 * (1 or 2 days) and filtering client-side.
 *
 * The backend caches per (UTC date, league, season), so adjacent local-day
 * tabs that need the same UTC date share a cache entry — no extra upstream
 * cost beyond the second HTTP round-trip when the timezone straddles UTC
 * midnight.
 */
export const fetchMatchesForLocalDate = async (
  localDate: Date,
  leagueId?: number,
  season?: number,
): Promise<Match[] | null> => {
  const utcDates = utcDateStringsForLocalDate(localDate);
  try {
    const results = await Promise.all(
      utcDates.map(utc => fetchMatchesByUtcDate(utc, leagueId, season)),
    );

    // Dedupe by fixture id (a game whose UTC date is e.g. Jun 17 will appear
    // in BOTH the Jun 16 *and* Jun 17 UTC responses for an ET user whose
    // local Jun 16 happens to include 01:00 UTC Jun 17; we only want it once).
    const merged = new Map<number, Match>();
    for (const arr of results) {
      if (!arr) continue;
      for (const m of arr) {
        if (m?.fixture?.id != null) {
          merged.set(m.fixture.id, m);
        }
      }
    }

    const filtered = Array.from(merged.values()).filter(m =>
      m?.fixture?.date ? isOnLocalDate(m.fixture.date, localDate) : false,
    );

    // Stable chronological order so cards render kickoff-ascending regardless
    // of which UTC fetch they came from.
    filtered.sort((a, b) => {
      const ta = new Date(a.fixture.date).getTime();
      const tb = new Date(b.fixture.date).getTime();
      return ta - tb;
    });

    return filtered;
  } catch (error) {
    console.error('Error fetching matches for local date:', error);
    return null;
  }
};

export const fetchLineup = async (
  matchId: number,
): Promise<LineupData | null> => {
  try {
    const data = await makeApiRequest(
      `/futbol/lineup?match_id=${matchId}`,
      'GET',
    );

    if (
      data.message === 'No lineup data available' ||
      data.message === 'No lineup data'
    ) {
      return null;
    }

    let lineupData = data;
    if (data.response) {
      lineupData = data.response;
    }

    if (!lineupData || typeof lineupData !== 'object') {
      console.warn('Invalid lineup response format:', data);
      return null;
    }

    if (!lineupData.home || !lineupData.away) {
      console.warn(
        'Missing home or away team data in lineup response:',
        lineupData,
      );
      return null;
    }

    if (!lineupData.home.starters) lineupData.home.starters = [];
    if (!lineupData.away.starters) lineupData.away.starters = [];
    if (!lineupData.home.substitutes) lineupData.home.substitutes = [];
    if (!lineupData.away.substitutes) lineupData.away.substitutes = [];

    if (
      !Array.isArray(lineupData.home.starters) ||
      !Array.isArray(lineupData.away.starters)
    ) {
      console.warn('Starters is not an array in lineup response:', lineupData);
      return null;
    }

    return lineupData;
  } catch (error) {
    console.error('Error fetching lineup:', error);
    return null;
  }
};

export const fetchStandings = async (
  leagueId: number,
  seasonYear: number,
): Promise<Standing[][]> => {
  try {
    const data = await makeApiRequest(
      `/futbol/league_standings?league_id=${leagueId}&season=${seasonYear}`,
      'GET',
    );
    if (data.response && data.response[0] && data.response[0].league) {
      return data.response[0].league.standings || [];
    }
    if (data.standings) return data.standings;
    if (Array.isArray(data)) return data;
    console.warn('Unexpected standings response format:', data);
    return [];
  } catch (error) {
    console.error('Error fetching standings:', error);
    throw error;
  }
};

export const createMatch = async (matchData: any): Promise<any> => {
  try {
    const data = await makeApiRequest('/futbol/matches', 'POST', {
      body: JSON.stringify(matchData),
    });
    return data;
  } catch (error) {
    console.error('Error creating match:', error);
    throw error;
  }
};

export const updateMatch = async (
  matchId: number,
  matchData: any,
): Promise<any> => {
  try {
    const data = await makeApiRequest(`/futbol/matches/${matchId}`, 'PUT', {
      body: JSON.stringify(matchData),
    });
    return data;
  } catch (error) {
    console.error('Error updating match:', error);
    throw error;
  }
};

export const deleteMatch = async (matchId: number): Promise<boolean> => {
  try {
    await makeApiRequest(`/futbol/matches/${matchId}`, 'DELETE');
    return true;
  } catch (error) {
    console.error('Error deleting match:', error);
    throw error;
  }
};
