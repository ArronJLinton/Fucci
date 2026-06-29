import {useEffect, useRef} from 'react';
import {useQueryClient} from '@tanstack/react-query';
import {useAuth} from '../context/AuthContext';
import {fetchDebatesFeed} from '../services/debate';
import {mainDebatesFeedQueryKey} from '../queries/keys';

const DEBATES_STALE_MS = 2 * 60 * 1000;

/** After sign-in, prefetch the personalized debates feed into React Query. */
export function useAuthCacheWarm(): void {
  const queryClient = useQueryClient();
  const {token, user, isLoggedIn, isReady} = useAuth();
  const warmedForToken = useRef<string | null>(null);

  useEffect(() => {
    if (!isReady || !isLoggedIn || !token) {
      warmedForToken.current = null;
      return;
    }
    if (warmedForToken.current === token) {
      return;
    }
    warmedForToken.current = token;

    void queryClient.prefetchQuery({
      queryKey: mainDebatesFeedQueryKey(token, user?.id),
      queryFn: async () => {
        const data = await fetchDebatesFeed(token, {
          new_limit: 30,
          voted_limit: 30,
        });
        return {kind: 'auth' as const, ...data};
      },
      staleTime: DEBATES_STALE_MS,
      gcTime: 10 * 60 * 1000,
    });
  }, [isReady, isLoggedIn, token, user?.id, queryClient]);
}
