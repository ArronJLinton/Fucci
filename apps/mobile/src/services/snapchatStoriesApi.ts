import {makeApiRequest} from './api';

/** Aligned with server Redis TTL (`internal/cache.SnapchatUserStoriesTTL` = 3m). */
export const SNAPCHAT_USER_STORIES_STALE_MS = 3 * 60 * 1000;

export const snapchatUserStoriesQueryKey = (username: string) =>
  ['snapchatUserStories', username] as const;

export type SnapchatUserStoriesResponse = {
  user?: {
    name?: string;
    username?: string;
    profile_picture_url?: string;
  };
  stories?: SnapchatStoryItem[];
  message?: string;
};

export type SnapchatStoryItem = {
  snapIndex: number;
  snapMediaType: number;
  snapUrls?: {
    mediaUrl?: string;
    mediaPreviewUrl?: { value?: string } | null;
  } | null;
  snapTitle?: string | null;
  user?: {
    title?: string;
    username?: string;
  };
};

export function fetchSnapchatUserStories(
  snapchatUsername: string,
): Promise<SnapchatUserStoriesResponse> {
  const q = new URLSearchParams({ username: snapchatUsername });
  return makeApiRequest(`/snapchat/stories?${q.toString()}`, 'GET');
}

/** True when the API returned at least one story with a usable media URL (same filter as the stories UI). */
export function hasRenderableSnapchatStories(
  res: SnapchatUserStoriesResponse | null | undefined,
): boolean {
  const stories = res?.stories ?? [];
  return stories.some(s => Boolean(s?.snapUrls?.mediaUrl));
}
