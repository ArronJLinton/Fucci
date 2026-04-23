import {YOUTUBE_DATA_API_KEY} from '../config/youtubeDataApiKey';

/**
 * YouTube Data API v3 – search. Docs:
 * https://developers.google.com/youtube/v3/docs/search/list
 *
 * Use `videoEmbeddable=true` (not `embedable`) and `type=video` or the request returns 400.
 */
const SEARCH_URL = 'https://www.googleapis.com/youtube/v3/search';

/** https://www.youtube.com/@FCBarcelona — matches YouTube "channel" / @handle (FC Barcelona official). */
export const FC_BARCELONA_CHANNEL_ID = 'UC14UlmYlSNiQCBe9Eookf_A';

export type YoutubeSearchListItem = {
  kind: string;
  id: {kind: string; videoId?: string; channelId?: string};
  snippet: {
    channelTitle: string;
    title: string;
    /** ISO 8601 from API */
    publishedAt: string;
  };
};

export type YoutubeSearchListResponse = {
  kind: string;
  items?: YoutubeSearchListItem[];
  error?: {code: number; message: string; errors?: {message: string}[]};
  pageInfo?: {totalResults: number; resultsPerPage: number};
};

export function isSearchVideoItem(
  item: YoutubeSearchListItem,
): item is YoutubeSearchListItem & {
  id: {kind: 'youtube#video'; videoId: string};
} {
  return 'videoId' in item.id && Boolean(item.id.videoId);
}

export type SearchVideosParams = {
  /**
   * Free-text search. Optional when `channelId` is set (channel-only listing still needs a
   * `q` for Shorts: we pass `#shorts` via `withShortsQuery` when `shortsOnly` is true).
   */
  q?: string;
  /**
   * Restricts results to a single channel (`search.list` `channelId` — see
   * https://www.youtube.com/@FCBarcelona/shorts → `FC_BARCELONA_CHANNEL_ID`).
   */
  channelId?: string;
  /** @default 'date' */
  order?:
    | 'date'
    | 'relevance'
    | 'viewCount'
    | 'rating'
    | 'title'
    | 'videoCount';
  maxResults?: number;
  /**
   * Bias results toward YouTube Shorts. The Data API has no `isShort` field; this sets
   * `videoDuration=short` (under 4 minutes) and appends `#shorts` to the query. See
   * https://developers.google.com/youtube/v3/docs/search/list — some non-Shorts can still appear.
   * @default true
   */
  shortsOnly?: boolean;
  /**
   * Only include videos published on this **local** calendar day (00:00–23:59:59.999).
   * Sets `publishedAfter` / `publishedBefore` and re-sorts results oldest → newest.
   * @default the current day when the request runs
   */
  publishedOnLocalDay?: Date;
};

function withShortsQuery(q: string): string {
  if (/\b#?shorts\b/i.test(q)) {
    return q.trim();
  }
  return `${q.trim()} #shorts`.trim();
}

/** Stable id for the local calendar day (cache keys, labels). */
export function formatLocalYmd(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(
    d.getDate(),
  ).padStart(2, '0')}`;
}

/** Local calendar day start/end in ms (for `publishedAt` filtering). */
export function getLocalDayBoundsMs(day: Date): {startMs: number; endMs: number} {
  const start = new Date(
    day.getFullYear(),
    day.getMonth(),
    day.getDate(),
    0,
    0,
    0,
    0,
  );
  const end = new Date(
    day.getFullYear(),
    day.getMonth(),
    day.getDate(),
    23,
    59,
    59,
    999,
  );
  return {startMs: start.getTime(), endMs: end.getTime()};
}

function todayLocalOrParam(day?: Date): Date {
  return day ? new Date(day) : new Date();
}

/**
 * Newest first from the API; we want oldest first for the chosen day, by `publishTime`.
 */
function filterSortTodayVideos(
  data: YoutubeSearchListResponse,
  day: Date,
): YoutubeSearchListResponse {
  const {startMs, endMs} = getLocalDayBoundsMs(day);
  const raw = data.items ?? [];
  const filtered = raw
    .filter(
      (item): item is YoutubeSearchListItem & {snippet: {publishedAt: string}} => {
        if (!isSearchVideoItem(item)) {
          return false;
        }
        const t = item.snippet?.publishedAt;
        if (typeof t !== 'string') {
          return false;
        }
        const ms = new Date(t).getTime();
        return ms >= startMs && ms <= endMs;
      },
    )
    .sort(
      (a, b) =>
        new Date(a.snippet.publishedAt).getTime() -
        new Date(b.snippet.publishedAt).getTime(),
    );
  return {...data, items: filtered};
}

export async function searchVideos(
  params: SearchVideosParams,
): Promise<YoutubeSearchListResponse> {
  const {
    q = '',
    channelId,
    order = 'date',
    maxResults = 10,
    shortsOnly = true,
    publishedOnLocalDay,
  } = params;

  if (!channelId && !q.trim() && !shortsOnly) {
    throw new Error('searchVideos: set `channelId` and/or `q`, or use shortsOnly for #shorts');
  }

  const day = todayLocalOrParam(publishedOnLocalDay);
  const {startMs, endMs} = getLocalDayBoundsMs(day);
  const publishedAfter = new Date(startMs).toISOString();
  const publishedBefore = new Date(endMs).toISOString();

  const qResolved = shortsOnly
    ? withShortsQuery(q)
    : q.trim();

  const sp = new URLSearchParams({
    part: 'snippet',
    key: YOUTUBE_DATA_API_KEY,
    q: qResolved,
    type: 'video',
    order,
    maxResults: String(Math.min(50, Math.max(1, maxResults))),
    videoEmbeddable: 'true',
    publishedAfter,
    publishedBefore,
  });

  if (channelId) {
    sp.set('channelId', channelId);
  }

  if (shortsOnly) {
    sp.set('videoDuration', 'short');
  }

  const res = await fetch(`${SEARCH_URL}?${sp.toString()}`);
  const data = (await res.json()) as YoutubeSearchListResponse;

  if (!res.ok) {
    const msg =
      data.error?.message ||
      data.error?.errors?.[0]?.message ||
      `HTTP ${res.status}`;
    throw new Error(msg);
  }

  return filterSortTodayVideos(data, day);
}
