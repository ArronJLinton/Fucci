import {makeApiRequest} from './api';

/** Aligned with server Redis TTL (`internal/cache.YouTubeShortsTTL` = 24h). */
export const MATCH_SHORTS_STALE_MS = 24 * 60 * 60 * 1000;

export const matchShortsQueryKey = (matchId: number) =>
  ['matchShorts', matchId] as const;

export type YouTubeShort = {
  video_id: string;
  title: string;
  thumbnail_url: string;
  embed_url: string;
  duration: string;
  published_at: string;
};

export type MatchShortsTeam = {
  lookup_key: string;
  has_shorts: boolean;
  shorts: YouTubeShort[];
};

export type MatchShortsResponse = {
  match_id: string;
  teams: {
    home: MatchShortsTeam;
    away: MatchShortsTeam;
  };
};

export function fetchMatchShorts(matchId: number): Promise<MatchShortsResponse> {
  return makeApiRequest(`/matches/${matchId}/stories/shorts`, 'GET');
}

export function hasTeamShorts(
  team: MatchShortsTeam | null | undefined,
): boolean {
  return Boolean(team?.has_shorts && (team.shorts?.length ?? 0) > 0);
}

/** Parse ISO 8601 YouTube duration (e.g. PT58S, PT1M30S) to seconds. */
export function parseYouTubeDurationSeconds(iso: string): number {
  const match = iso.match(/PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?/);
  const hours = parseInt(match?.[1] ?? '0', 10);
  const minutes = parseInt(match?.[2] ?? '0', 10);
  const seconds = parseInt(match?.[3] ?? '0', 10);
  return hours * 3600 + minutes * 60 + seconds;
}

/** Origin used as WebView baseUrl — matches react-native-youtube-iframe hosted player (valid Referer). */
export const YOUTUBE_SHORT_PLAYER_BASE_URL =
  'https://lonelycpp.github.io/react-native-youtube-iframe/';

/**
 * Full-viewport YouTube IFrame API page for vertical Shorts (no 16:9 letterbox hack).
 */
export function youtubeShortPlayerHtml(videoId: string): string {
  const safeId = videoId.replace(/[^a-zA-Z0-9_-]/g, '');
  return `<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no" />
    <meta name="referrer" content="strict-origin-when-cross-origin" />
    <style>
      html, body {
        margin: 0;
        padding: 0;
        width: 100%;
        height: 100%;
        background: #000;
        overflow: hidden;
      }
      #player {
        position: fixed;
        inset: 0;
        width: 100%;
        height: 100%;
      }
    </style>
  </head>
  <body>
    <div id="player"></div>
    <script>
      var tag = document.createElement('script');
      tag.src = 'https://www.youtube.com/iframe_api';
      document.head.appendChild(tag);
      var player;
      function onYouTubeIframeAPIReady() {
        var w = window.innerWidth;
        var h = window.innerHeight;
        player = new YT.Player('player', {
          width: w,
          height: h,
          videoId: '${safeId}',
          playerVars: {
            autoplay: 1,
            controls: 0,
            playsinline: 1,
            rel: 0,
            modestbranding: 1,
            fs: 0,
            iv_load_policy: 3,
            disablekb: 1,
          },
          events: {
            onReady: function () {
              window.ReactNativeWebView.postMessage(JSON.stringify({ type: 'ready' }));
              player.playVideo();
            },
            onStateChange: function (e) {
              if (e.data === YT.PlayerState.ENDED) {
                window.ReactNativeWebView.postMessage(JSON.stringify({ type: 'ended' }));
              }
            },
            onError: function (e) {
              window.ReactNativeWebView.postMessage(JSON.stringify({ type: 'error', code: e.data }));
            },
          },
        });
      }
    </script>
  </body>
</html>`;
}
