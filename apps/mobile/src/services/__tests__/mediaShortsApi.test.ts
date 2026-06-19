/// <reference types="jest" />

import {
  fetchMediaShorts,
  mediaShortsQueryKey,
  type MediaOutletShorts,
} from '../mediaShortsApi';
import * as api from '../api';

describe('mediaShortsApi', () => {
  describe('mediaShortsQueryKey', () => {
    it('uses a stable query key', () => {
      expect(mediaShortsQueryKey).toEqual(['mediaShorts']);
    });
  });

  describe('fetchMediaShorts', () => {
    afterEach(() => {
      jest.restoreAllMocks();
    });

    it('calls the news media stories endpoint', async () => {
      const outlet: MediaOutletShorts = {
        lookup_key: 'fox_soccer',
        display_name: 'FOX SPORTS',
        has_shorts: true,
        thumbnail_url: 'https://img/fox.jpg',
        shorts: [
          {
            video_id: 'abc',
            title: 'Goal',
            thumbnail_url: 'https://img/fox.jpg',
            embed_url: 'https://youtube.com/embed/abc',
            duration: 'PT30S',
            published_at: '2026-06-19T00:00:00Z',
          },
        ],
      };

      jest.spyOn(api, 'makeApiRequest').mockResolvedValue({outlets: [outlet]});

      const res = await fetchMediaShorts();
      expect(api.makeApiRequest).toHaveBeenCalledWith(
        '/news/stories/shorts',
        'GET',
      );
      expect(res.outlets[0].display_name).toBe('FOX SPORTS');
      expect(res.outlets[0].has_shorts).toBe(true);
    });
  });
});
