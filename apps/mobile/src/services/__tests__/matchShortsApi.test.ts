/// <reference types="jest" />

import {
  fetchMatchShorts,
  hasTeamShorts,
  matchShortsQueryKey,
  parseYouTubeDurationSeconds,
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

  describe('youtubeShortPlayerHtml', () => {
    it('embeds sanitized video id and iframe API hooks', () => {
      const html = youtubeShortPlayerHtml('Jf3zdhWWfe4');
      expect(html).toContain("videoId: 'Jf3zdhWWfe4'");
      expect(html).toContain('youtube.com/iframe_api');
      expect(html).toContain("type: 'ready'");
      expect(html).toContain("type: 'ended'");
      expect(html).toContain('autoplay: 1');
      expect(html).toContain('controls: 0');
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
