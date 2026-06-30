import {queryClient} from '../config/queryClient';
import {warmAppCache} from './warmAppCache';

/**
 * Runs before first paint (while native splash is held).
 * Seeds React Query with news, matches, team Shorts, media Shorts, and guest debates.
 */
export async function prepareApp(): Promise<void> {
  await warmAppCache(queryClient);
}
