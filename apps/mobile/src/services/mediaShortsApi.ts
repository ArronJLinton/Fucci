import {makeApiRequest} from './api';
import type {YouTubeShort} from './matchShortsApi';

/** Aligned with server Redis TTL (`internal/cache.YouTubeShortsTTL` = 24h). */
export const MEDIA_SHORTS_STALE_MS = 24 * 60 * 60 * 1000;

export const mediaShortsQueryKey = ['mediaShorts'] as const;

export type MediaOutletShorts = {
  lookup_key: string;
  display_name: string;
  has_shorts: boolean;
  thumbnail_url: string;
  shorts: YouTubeShort[];
};

export type MediaShortsResponse = {
  outlets: MediaOutletShorts[];
};

export function fetchMediaShorts(): Promise<MediaShortsResponse> {
  return makeApiRequest('/news/stories/shorts', 'GET');
}
