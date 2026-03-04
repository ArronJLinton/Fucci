/**
 * News feature type definitions
 *
 * These types match the backend API response structure for news articles
 */

/**
 * News article from the API
 */
export interface NewsArticle {
  id: string;
  title: string;
  snippet?: string;
  imageUrl?: string;
  sourceUrl: string;
  sourceName: string;
  publishedAt: string; // ISO 8601 format
  relativeTime: string; // "2 hours ago", "1 day ago"
}

/**
 * News API response structure
 */
export interface NewsAPIResponse {
  todayArticles: NewsArticle[];
  historyArticles: NewsArticle[];
  cached: boolean;
  cachedAt?: string; // ISO 8601 format; may be set for both cached and fresh responses
  // Deprecated: Use todayArticles and historyArticles instead
  articles?: NewsArticle[];
}

/**
 * Match-specific news API response structure
 */
export interface MatchNewsAPIResponse {
  articles: NewsArticle[];
  cached: boolean;
  cachedAt?: string; // ISO 8601 format; may be set for both cached and fresh responses
}
