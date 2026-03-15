import {useState, useCallback} from 'react';

/**
 * Hook for pull-to-refresh UX. Returns refreshing state and an onRefresh handler
 * that runs the given async callback and clears refreshing when done.
 * Reusable across any screen that has a refresh/retry action.
 */
export function usePullToRefresh(
  refreshFn: () => Promise<void>,
): {refreshing: boolean; onRefresh: () => Promise<void>} {
  const [refreshing, setRefreshing] = useState(false);

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await refreshFn();
    } finally {
      setRefreshing(false);
    }
  }, [refreshFn]);

  return {refreshing, onRefresh};
}
