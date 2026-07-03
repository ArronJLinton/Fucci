/// <reference types="jest" />

import {
  buildStorySlides,
  fetchMatchShorts,
  hasTeamShorts,
  teamHasStoryContent,
  matchShortsQueryKey,
  parseYouTubeDurationSeconds,
  youtubePlayerContainLayout,
  youtubePlayerCoverLayout,
  youtubePlayerShortsLayout,
  youtubeShortPlayerHtml,
  YOUTUBE_SHORT_PLAYER_BASE_URL,
  type MatchShortsTeam,
} from '../matchShortsApi';
import * as api from '../api';

describe('matchShortsApi', () => {
  describe('matchShortsQueryKey', () => {
    it('keys by match fixture id', () => {
      expect(matchShortsQueryKey(1489391)).toEqual(['matchShorts', 1489391]);
    });
  });

  describe('parseYouTubeDurationSeconds', () => {
    it('parses hours, minutes, and seconds', () => {
      expect(parseYouTubeDurationSeconds('PT1H2M3S')).toBe(3723);
      expect(parseYouTubeDurationSeconds('PT58S')).toBe(58);
      expect(parseYouTubeDurationSeconds('PT1M30S')).toBe(90);
      expect(parseYouTubeDurationSeconds('')).toBe(0);
    });
  });

  describe('hasTeamShorts', () => {
    const withShorts: MatchShortsTeam = {
      lookup_key: 'usa',
      has_shorts: true,
      shorts: [
        {
          video_id: 'abc',
          title: 'T',
          thumbnail_url: '',
          embed_url: 'https://youtube.com/embed/abc',
          duration: 'PT30S',
          published_at: '2026-06-19T00:00:00Z',
        },
      ],
      has_user_stories: false,
      user_stories: [],
    };

    it('is true when flag and shorts list are present', () => {
      expect(hasTeamShorts(withShorts)).toBe(true);
    });

    it('is false when has_shorts is false', () => {
      expect(hasTeamShorts({...withShorts, has_shorts: false})).toBe(false);
    });

    it('is false when shorts list is empty', () => {
      expect(hasTeamShorts({...withShorts, shorts: []})).toBe(false);
    });

    it('is false for null/undefined', () => {
      expect(hasTeamShorts(null)).toBe(false);
      expect(hasTeamShorts(undefined)).toBe(false);
    });
  });

  describe('teamHasStoryContent', () => {
    it('is true when user stories exist', () => {
      expect(
        teamHasStoryContent({
          lookup_key: 'spain',
          has_shorts: false,
          shorts: [],
          has_user_stories: true,
          user_stories: [
            {
              id: '1',
              content_type: 'photo',
              media_url: 'https://example.com/p.jpg',
              user_id: 1,
              created_at: '2026-07-02T00:00:00Z',
            },
          ],
        }),
      ).toBe(true);
    });
  });

  describe('buildStorySlides', () => {
    it('orders fan stories before youtube shorts', () => {
      const slides = buildStorySlides(
        [
          {
            id: 'fan-1',
            content_type: 'photo',
            media_url: 'https://example.com/p.jpg',
            user_id: 2,
            created_at: '2026-07-02T00:00:00Z',
          },
        ],
        [
          {
            video_id: 'yt-1',
            title: 'Highlight',
            thumbnail_url: '',
            embed_url: '',
            duration: 'PT30S',
            published_at: '2026-07-02T00:00:00Z',
          },
        ],
      );
      expect(slides).toHaveLength(2);
      expect(slides[0].kind).toBe('fan');
      expect(slides[1].kind).toBe('youtube');
    });
  });

  describe('youtubePlayerContainLayout', () => {
    it('fits the full Short frame in portrait without cropping', () => {
      const baseHeight = Math.round(360 / (9 / 16));
      const layout = youtubePlayerContainLayout(393, 852);
      expect(layout.scale).toBe(Math.min(393 / 360, 852 / baseHeight));
    });

    it('fits the full Short frame in landscape without cropping', () => {
      const baseHeight = Math.round(360 / (9 / 16));
      const layout = youtubePlayerContainLayout(852, 393);
      expect(layout.scale).toBe(Math.min(852 / 360, 393 / baseHeight));
    });
  });

  describe('youtubePlayerShortsLayout', () => {
    it('uses cover-fit in landscape for full screen', () => {
      const baseHeight = Math.round(360 / (9 / 16));
      const layout = youtubePlayerShortsLayout(852, 393);
      expect(layout.isLandscape).toBe(true);
      expect(layout.scale).toBe(Math.max(852 / 360, 393 / baseHeight));
    });

    it('uses contain-fit in portrait to avoid cropping', () => {
      const baseHeight = Math.round(360 / (9 / 16));
      const layout = youtubePlayerShortsLayout(393, 852);
      expect(layout.isLandscape).toBe(false);
      expect(layout.scale).toBe(Math.min(393 / 360, 852 / baseHeight));
    });
  });

  describe('youtubePlayerCoverLayout', () => {
    it('scales up to cover landscape viewport', () => {
      const layout = youtubePlayerCoverLayout(852, 393, 9 / 16, 0);
      expect(layout.baseWidth).toBe(360);
      expect(layout.scale).toBeGreaterThanOrEqual(852 / 360);
      expect(layout.baseHeight).toBe(Math.round(360 / (9 / 16)));
    });

    it('applies extra zoom when uiCropRatio is set', () => {
      const noCrop = youtubePlayerCoverLayout(393, 852, 9 / 16, 0);
      const cropped = youtubePlayerCoverLayout(393, 852, 9 / 16, 0.14);
      expect(cropped.scale).toBeGreaterThan(noCrop.scale);
    });
  });

  describe('youtubeShortPlayerHtml', () => {
    it('embeds sanitized video id and iframe API hooks', () => {
      const html = youtubeShortPlayerHtml('Jf3zdhWWfe4');
      expect(html).toContain("videoId: 'Jf3zdhWWfe4'");
      expect(html).toContain('youtube.com/iframe_api');
      expect(html).toContain("post('playing')");
      expect(html).toContain("post('ended')");
      expect(html).toContain('autoplay: 1');
      expect(html).not.toContain('mute: 1');
      expect(html).toContain('controls: 0');
      expect(html).toContain("transform = 'scale('");
      expect(html).toContain('resizeYouTubePlayer');
      expect(html).toContain('isLandscape');
      expect(html).toContain('Math.max(vw / baseW, vh / baseH)');
      expect(html).toContain('Math.min(vw / baseW, vh / baseH)');
      expect(html).toContain('fs: 0');
    });

    it('strips unsafe characters from video id', () => {
      const html = youtubeShortPlayerHtml("bad<script>'");
      expect(html).not.toContain("videoId: 'bad<script>'");
      expect(html).toContain("videoId: 'badscript'");
    });
  });

  describe('fetchMatchShorts', () => {
    afterEach(() => {
      jest.restoreAllMocks();
    });

    it('calls the match stories endpoint', async () => {
      const makeApiRequest = jest.spyOn(api, 'makeApiRequest').mockResolvedValue({
        match_id: '1489391',
        teams: {
          home: {lookup_key: 'united states', has_shorts: true, shorts: []},
          away: {lookup_key: 'australia', has_shorts: false, shorts: []},
        },
      });

      const res = await fetchMatchShorts(1489391);
      expect(makeApiRequest).toHaveBeenCalledWith(
        '/matches/1489391/stories/shorts',
        'GET',
      );
      expect(res.match_id).toBe('1489391');
      expect(res.teams.home.lookup_key).toBe('united states');
    });
  });

  it('exports hosted player base URL for WebView referer', () => {
    expect(YOUTUBE_SHORT_PLAYER_BASE_URL).toContain('lonelycpp.github.io');
  });
});
