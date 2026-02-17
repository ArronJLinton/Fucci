import {apiConfig} from '../config/environment';
import type {NewsAPIResponse, MatchNewsAPIResponse} from '../types/news';

/**
 * News API service client
 * Fetches football news from the backend API
 */

/**
 * Fetches football news articles from the backend API
 * @returns Promise resolving to NewsAPIResponse
 * @throws Error if the API request fails
 */
export const fetchFootballNews = async (): Promise<NewsAPIResponse> => {
  const url = `${apiConfig.baseURL}/news/football`;

  try {
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        ...apiConfig.headers,
      },
    });

    if (!response.ok) {
      const errorText = await response.text();
      console.error('[fetchFootballNews] Error response:', errorText);
      throw new Error(
        `API request failed: ${response.status} ${response.statusText}`,
      );
    }

    const data: NewsAPIResponse = await response.json();
    return data;
  } catch (error) {
    console.error(`[fetchFootballNews] Request failed for ${url}:`, error);
    if (error instanceof TypeError && error.message.includes('fetch')) {
      throw new Error(
        'Network error: Unable to connect to the server. Please check if the backend is running.',
      );
    }
    throw error;
  }
};

/**
 * Fetches match-specific news articles from the backend API
 * @param homeTeam - Name of the home team
 * @param awayTeam - Name of the away team
 * @param matchId - Match ID for caching
 * @param matchStatus - Match status (e.g. NS, 1H, FT) - used for time filtering
 * @param matchEndTime - ISO8601 string of match end time (for completed matches only)
 * @returns Promise resolving to MatchNewsAPIResponse
 * @throws Error if the API request fails
 */
export const fetchMatchNews = async (
  homeTeam: string,
  awayTeam: string,
  matchId: string,
  matchStatus: string,
  matchEndTime: string,
): Promise<MatchNewsAPIResponse> => {
  const params = new URLSearchParams({
    homeTeam,
    awayTeam,
    matchId,
    matchStatus,
  });
  if (matchEndTime) {
    params.append('matchEndTime', matchEndTime);
  }
  const url = `${apiConfig.baseURL}/news/football/match?${params.toString()}`;

  try {
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        ...apiConfig.headers,
      },
    });

    if (!response.ok) {
      const errorText = await response.text();
      console.error('[fetchMatchNews] Error response:', errorText);
      throw new Error(
        `API request failed: ${response.status} ${response.statusText}`,
      );
    }

    const data: MatchNewsAPIResponse = await response.json();
    return data;
  } catch (error) {
    console.error(`[fetchMatchNews] Request failed for ${url}:`, error);
    if (error instanceof TypeError && error.message.includes('fetch')) {
      throw new Error(
        'Network error: Unable to connect to the server. Please check if the backend is running.',
      );
    }
    throw error;
  }
};
