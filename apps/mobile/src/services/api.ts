// Import types
import {DebateResponse} from '../types/debate';
import {apiConfig} from '../config/environment';

// Types
interface Standing {
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

interface Match {
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

interface LineupData {
  home: {
    starters: any[];
    substitutes: any[];
  };
  away: {
    starters: any[];
    substitutes: any[];
  };
}

// Helper function for making API requests
const makeApiRequest = async (
  endpoint: string,
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH' = 'GET',
  options: RequestInit = {},
) => {
  const url = `${apiConfig.baseURL}${endpoint}`;
  try {
    const response = await fetch(url, {
      method,
      redirect: 'follow',
      headers: {
        ...apiConfig.headers,
        ...options.headers,
      },
      ...options,
    });
    if (!response.ok) {
      throw new Error(
        `API request failed: ${response.status} ${response.statusText}`,
      );
    }

    return response.json();
  } catch (error) {
    console.error(`API request failed for ${url}:`, error);
    throw error;
  }
};

// Futbol API Functions
export const fetchMatches = async (
  date: Date,
  leagueId?: number,
): Promise<Match[] | null> => {
  try {
    const formattedDate = `${date.getFullYear()}-${String(
      date.getMonth() + 1,
    ).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;

    let endpoint = `/futbol/matches?date=${formattedDate}`;
    if (leagueId) {
      endpoint += `&league_id=${leagueId}`;
    }

    const data = await makeApiRequest(endpoint, 'GET');
    return data.response;
  } catch (error) {
    console.error('Error fetching matches:', error);
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

    // Check for no lineup data message
    if (
      data.message === 'No lineup data available' ||
      data.message === 'No lineup data'
    ) {
      return null;
    }

    // Handle different response structures
    let lineupData = data;
    if (data.response) {
      lineupData = data.response;
    }

    // Validate the data structure
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

    // Ensure starters arrays exist and are arrays
    if (!lineupData.home.starters) {
      lineupData.home.starters = [];
    }
    if (!lineupData.away.starters) {
      lineupData.away.starters = [];
    }
    if (!lineupData.home.substitutes) {
      lineupData.home.substitutes = [];
    }
    if (!lineupData.away.substitutes) {
      lineupData.away.substitutes = [];
    }

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
    return null; // Return null instead of throwing to prevent crashes
  }
};

export const fetchDebate = async (
  matchId: number,
  type: string = 'pre_match',
): Promise<DebateResponse | null> => {
  try {
    const data = await makeApiRequest(
      `/debates/generate?match_id=${matchId}&type=${type}`,
      'POST',
    );

    // Handle different response structures
    if (data.info && data.info.includes('cannot generate')) {
      console.log('Debate generation info:', data.info);
      return null;
    }

    if (data.message && data.message.includes('No debate data')) {
      return null;
    }

    return data;
  } catch (error) {
    console.error('Error fetching debate:', error);
    return null; // Return null instead of throwing to prevent crashes
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
    // Handle different response structures
    if (data.response && data.response[0] && data.response[0].league) {
      return data.response[0].league.standings || [];
    } else if (data.standings) {
      return data.standings;
    } else if (Array.isArray(data)) {
      return data;
    } else {
      console.warn('Unexpected standings response format:', data);
      return [];
    }
  } catch (error) {
    console.error('Error fetching standings:', error);
    throw error;
  }
};

// Example utility functions demonstrating different HTTP methods
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
