import {useQuery, useQueryClient} from '@tanstack/react-query';
import {fetchMatchNews} from '../services/newsService';
import type {MatchNewsAPIResponse} from '../types/news';

/**
 * Custom hook for fetching and managing match-specific news data
 * Uses React Query for caching and state management
 *
 * @param homeTeam - Name of the home team
 * @param awayTeam - Name of the away team
 * @param matchId - Match ID for caching
 * @param matchStatus - Match status (e.g. NS, 1H, FT) for time filtering
 * @param matchEndTime - ISO8601 string of match end time (for completed matches)
 * @returns Object containing articles, loading state, error, and refresh function
 */
export function useMatchNews(
  homeTeam: string,
  awayTeam: string,
  matchId: string,
  matchStatus: string,
  matchEndTime: string,
) {
  const queryClient = useQueryClient();

  const {data, isLoading, error, refetch, isRefetching} =
    useQuery<MatchNewsAPIResponse>({
      queryKey: ['news', 'match', matchId, homeTeam, awayTeam, matchStatus, matchEndTime],
      queryFn: async (): Promise<MatchNewsAPIResponse> => {
        try {
          console.log('[useMatchNews] Fetching match news...', {
            homeTeam,
            awayTeam,
            matchId,
            matchStatus,
          });
          const result = await fetchMatchNews(
            homeTeam,
            awayTeam,
            matchId,
            matchStatus,
            matchEndTime,
          );
          return result;
        } catch (err) {
          console.error('[useMatchNews] Error fetching match news:', err);
          throw err;
        }
      },
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 15 * 60 * 1000, // 15 minutes (formerly cacheTime)
      retry: 3,
      retryDelay: attemptIndex => Math.min(1000 * 2 ** attemptIndex, 30000),
      enabled: !!homeTeam && !!awayTeam && !!matchId, // Only fetch if all params are provided
    });

  /**
   * Clear the match news cache and optionally refetch
   * @param refetchAfterClear - If true, refetches data after clearing cache
   */
  const clearCache = async (refetchAfterClear: boolean = false) => {
    queryClient.removeQueries({
      queryKey: ['news', 'match', matchId],
    });
    console.log('[useMatchNews] Cleared match news cache');
    if (refetchAfterClear) {
      await refetch();
    }
  };

  /**
   * Invalidate the match news cache (marks as stale and triggers refetch)
   */
  const invalidateCache = () => {
    queryClient.invalidateQueries({
      queryKey: ['news', 'match', matchId],
    });
    console.log('[useMatchNews] Invalidated match news cache');
  };

  const matchNewsData: MatchNewsAPIResponse | undefined = data;

  return {
    articles: matchNewsData?.articles || [],
    loading: isLoading,
    error: error as Error | null,
    refresh: refetch,
    refreshing: isRefetching,
    cached: matchNewsData?.cached || false,
    cachedAt: matchNewsData?.cachedAt,
    clearCache,
    invalidateCache,
  };
}
