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

export type FanStory = {
  id: string;
  content_type: 'photo' | 'video';
  media_url: string;
  user_id: number;
  display_name?: string;
  avatar_url?: string;
  created_at: string;
};

export type MatchShortsTeam = {
  lookup_key: string;
  has_shorts: boolean;
  shorts: YouTubeShort[];
  has_user_stories: boolean;
  user_stories: FanStory[];
};

export type StorySlide =
  | {kind: 'fan'; story: FanStory; slideKey: string; durationMs: number}
  | {kind: 'youtube'; short: YouTubeShort; slideKey: string; durationMs: number};

export const FAN_STORY_PHOTO_DURATION_MS = 5000;
export const FAN_STORY_VIDEO_MAX_DURATION_MS = 60_000;

/**
 * Expo ImagePicker reports `asset.duration` in milliseconds for library videos.
 * Reject clips longer than the fan-story max (with a small picker tolerance).
 */
export function isLibraryVideoTooLong(
  durationMs: number | null | undefined,
  maxMs: number = FAN_STORY_VIDEO_MAX_DURATION_MS,
): boolean {
  return durationMs != null && durationMs > maxMs + 500;
}

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

export function hasTeamUserStories(
  team: MatchShortsTeam | null | undefined,
): boolean {
  return Boolean(team?.has_user_stories && (team.user_stories?.length ?? 0) > 0);
}

export function teamHasStoryContent(
  team: MatchShortsTeam | null | undefined,
): boolean {
  return hasTeamShorts(team) || hasTeamUserStories(team);
}

export function buildStorySlides(
  userStories: FanStory[] = [],
  youtubeShorts: YouTubeShort[] = [],
): StorySlide[] {
  const fanSlides: StorySlide[] = userStories.map(story => ({
    kind: 'fan' as const,
    story,
    slideKey: `fan-${story.id}`,
    durationMs:
      story.content_type === 'video'
        ? FAN_STORY_VIDEO_MAX_DURATION_MS
        : FAN_STORY_PHOTO_DURATION_MS,
  }));
  const ytSlides: StorySlide[] = youtubeShorts.map(short => ({
    kind: 'youtube' as const,
    short,
    slideKey: `yt-${short.video_id}`,
    durationMs: Math.max(parseYouTubeDurationSeconds(short.duration) * 1000, 3000),
  }));
  return [...fanSlides, ...ytSlides];
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

/** 9:16 — YouTube Shorts vertical aspect. */
export const YOUTUBE_SHORT_VIDEO_ASPECT = 9 / 16;

/**
 * Optional extra zoom after fit to trim Shorts iframe chrome (title, share).
 * Set to 0 — chrome crop was clipping video content; YouTube has no hide API.
 */
export const YOUTUBE_SHORT_UI_CROP_RATIO = 0;

/** Contain-fit layout — full Short frame visible, no edge crop. */
export function youtubePlayerContainLayout(
  viewportWidth: number,
  viewportHeight: number,
  videoAspect: number = YOUTUBE_SHORT_VIDEO_ASPECT,
): {
  baseWidth: number;
  baseHeight: number;
  scale: number;
} {
  const vw = Math.max(viewportWidth, 1);
  const vh = Math.max(viewportHeight, 1);
  const baseWidth = 360;
  const baseHeight = Math.round(baseWidth / videoAspect);
  const scale = Math.min(vw / baseWidth, vh / baseHeight);
  return {baseWidth, baseHeight, scale};
}

/** Scale to cover viewport (fills screen; may crop edges). */
export function youtubePlayerCoverLayout(
  viewportWidth: number,
  viewportHeight: number,
  videoAspect: number = YOUTUBE_SHORT_VIDEO_ASPECT,
  uiCropRatio: number = YOUTUBE_SHORT_UI_CROP_RATIO,
): {
  baseWidth: number;
  baseHeight: number;
  scale: number;
} {
  const vw = Math.max(viewportWidth, 1);
  const vh = Math.max(viewportHeight, 1);
  const baseWidth = 360;
  const baseHeight = Math.round(baseWidth / videoAspect);
  const coverScale = Math.max(vw / baseWidth, vh / baseHeight);
  const crop = Math.min(Math.max(uiCropRatio, 0), 0.24);
  const chromeZoom = crop > 0 ? 1 / (1 - 2 * crop) : 1;
  return {baseWidth, baseHeight, scale: coverScale * chromeZoom};
}

/** Cover in landscape, contain in portrait — fullscreen rotate without portrait crop. */
export function youtubePlayerShortsLayout(
  viewportWidth: number,
  viewportHeight: number,
  videoAspect: number = YOUTUBE_SHORT_VIDEO_ASPECT,
  uiCropRatio: number = YOUTUBE_SHORT_UI_CROP_RATIO,
): {
  baseWidth: number;
  baseHeight: number;
  scale: number;
  isLandscape: boolean;
} {
  const vw = Math.max(viewportWidth, 1);
  const vh = Math.max(viewportHeight, 1);
  const isLandscape = vw > vh;
  const baseWidth = 360;
  const baseHeight = Math.round(baseWidth / videoAspect);
  const fitScale = isLandscape
    ? Math.max(vw / baseWidth, vh / baseHeight)
    : Math.min(vw / baseWidth, vh / baseHeight);
  const crop = Math.min(Math.max(uiCropRatio, 0), 0.24);
  const chromeZoom = crop > 0 ? 1 / (1 - 2 * crop) : 1;
  return {
    baseWidth,
    baseHeight,
    scale: fitScale * chromeZoom,
    isLandscape,
  };
}

/**
 * Full-viewport YouTube IFrame API page for vertical Shorts.
 * Cover-fit in landscape (full screen); contain-fit in portrait (no video crop).
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
        left: 50%;
        top: 50%;
        transform-origin: center center;
        overflow: hidden;
      }
      #player iframe {
        width: 100% !important;
        height: 100% !important;
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
      function post(type, extra) {
        var payload = { type: type };
        if (extra) {
          for (var key in extra) {
            if (Object.prototype.hasOwnProperty.call(extra, key)) {
              payload[key] = extra[key];
            }
          }
        }
        window.ReactNativeWebView.postMessage(JSON.stringify(payload));
      }
      function startPlayback() {
        if (!player || !player.playVideo) return;
        try {
          if (player.unMute) player.unMute();
          if (player.setVolume) player.setVolume(100);
          player.playVideo();
        } catch (e) {}
      }
      window.startYouTubePlayback = startPlayback;
      var VIDEO_ASPECT = ${YOUTUBE_SHORT_VIDEO_ASPECT};
      var UI_CROP = ${YOUTUBE_SHORT_UI_CROP_RATIO};
      function resizePlayer(optW, optH) {
        var vw = (typeof optW === 'number' && optW > 0) ? optW : window.innerWidth;
        var vh = (typeof optH === 'number' && optH > 0) ? optH : window.innerHeight;
        var baseW = 360;
        var baseH = Math.round(baseW / VIDEO_ASPECT);
        var isLandscape = vw > vh;
        var fitScale = isLandscape
          ? Math.max(vw / baseW, vh / baseH)
          : Math.min(vw / baseW, vh / baseH);
        var crop = Math.min(Math.max(UI_CROP, 0), 0.24);
        var scale = fitScale * (crop > 0 ? 1 / (1 - 2 * crop) : 1);
        var el = document.getElementById('player');
        if (el) {
          el.style.width = baseW + 'px';
          el.style.height = baseH + 'px';
          el.style.marginLeft = (-baseW / 2) + 'px';
          el.style.marginTop = (-baseH / 2) + 'px';
          el.style.transform = 'scale(' + scale + ')';
        }
        if (player && player.setSize) {
          try {
            player.setSize(baseW, baseH);
          } catch (e) {}
        }
      }
      window.resizeYouTubePlayer = resizePlayer;
      window.addEventListener('resize', resizePlayer);
      window.addEventListener('orientationchange', function () {
        setTimeout(function () { resizePlayer(); }, 150);
        setTimeout(function () { resizePlayer(); }, 400);
      });
      if (window.visualViewport) {
        window.visualViewport.addEventListener('resize', function () {
          resizePlayer(window.visualViewport.width, window.visualViewport.height);
        });
      }
      function onYouTubeIframeAPIReady() {
        player = new YT.Player('player', {
          width: 360,
          height: Math.round(360 / VIDEO_ASPECT),
          videoId: '${safeId}',
          playerVars: {
            autoplay: 1,
            controls: 0,
            playsinline: 1,
            rel: 0,
            modestbranding: 1,
            fs: 0,
            iv_load_policy: 3,
            cc_load_policy: 0,
            disablekb: 1,
            origin: window.location.origin,
          },
          events: {
            onReady: function () {
              resizePlayer();
              startPlayback();
              setTimeout(startPlayback, 200);
              setTimeout(startPlayback, 600);
            },
            onStateChange: function (e) {
              if (e.data === YT.PlayerState.PLAYING) {
                post('playing');
              }
              if (
                e.data === YT.PlayerState.CUED ||
                e.data === YT.PlayerState.UNSTARTED
              ) {
                startPlayback();
              }
              if (e.data === YT.PlayerState.ENDED) {
                post('ended');
              }
            },
            onError: function (e) {
              post('error', { code: e.data });
            },
          },
        });
      }
    </script>
  </body>
</html>`;
}
