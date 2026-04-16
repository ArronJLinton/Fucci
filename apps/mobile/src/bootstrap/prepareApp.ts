import {fetchDebatesPublicFeed} from '../services/debate';
import {fetchFootballNews} from '../services/newsService';

/**
 * Runs before first paint (while native splash is held).
 * Extend with expo-font / asset preloads when needed.
 */
export async function prepareApp(): Promise<void> {
  await Promise.all([
    fetchFootballNews().catch(err => {
      console.warn('[prepareApp] football news prefetch failed:', err);
    }),
    fetchDebatesPublicFeed(12).catch(err => {
      console.warn('[prepareApp] debates public-feed prefetch failed:', err);
    }),
  ]);
}
