import {useQuery} from '@tanstack/react-query';
import {
  fetchMediaShorts,
  mediaShortsQueryKey,
  MEDIA_SHORTS_STALE_MS,
} from '../services/mediaShortsApi';

export function useMediaShorts() {
  return useQuery({
    queryKey: mediaShortsQueryKey,
    queryFn: fetchMediaShorts,
    staleTime: MEDIA_SHORTS_STALE_MS,
    gcTime: MEDIA_SHORTS_STALE_MS,
  });
}
