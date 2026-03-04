import {QueryClient} from '@tanstack/react-query';

/**
 * React Query client configuration for news feed caching
 *
 * Cache Strategy:
 * - staleTime: 5 minutes - Data is considered fresh for 5 minutes
 * - gcTime: 15 minutes - Cached data is kept for 15 minutes (matches backend Redis TTL)
 * - retry: 3 attempts with exponential backoff
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 15 * 60 * 1000, // 15 minutes
      retry: 3,
      retryDelay: attemptIndex => Math.min(1000 * 2 ** attemptIndex, 30000),
    },
  },
});

/**
 * Utility functions for clearing React Query cache
 */

/**
 * Clear all cached queries
 */
export const clearAllCache = () => {
  queryClient.clear();
  console.log('[Cache] Cleared all React Query cache');
};

/**
 * Invalidate and refetch news queries
 * This marks the data as stale and triggers a refetch
 */
export const invalidateNewsCache = () => {
  queryClient.invalidateQueries({queryKey: ['news', 'football']});
  console.log('[Cache] Invalidated news cache');
};

/**
 * Remove news queries from cache completely
 * This removes the data without refetching
 */
export const removeNewsCache = () => {
  queryClient.removeQueries({queryKey: ['news', 'football']});
  console.log('[Cache] Removed news cache');
};

/**
 * Reset all queries to their initial state
 */
export const resetAllQueries = () => {
  queryClient.resetQueries();
  console.log('[Cache] Reset all queries');
};
