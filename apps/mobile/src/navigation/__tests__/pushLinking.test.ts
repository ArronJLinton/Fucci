import {
  buildMatchFromDebate,
  buildPlaceholderMatchFromId,
  normalizePushNotificationData,
  parseMatchId,
  resolvePushNavigation,
  type PushNotificationData,
} from '../pushLinking';

describe('normalizePushNotificationData', () => {
  it('reads nested params from API payload', () => {
    expect(
      normalizePushNotificationData({
        type: 'news',
        route: 'NewsWebView',
        params: {url: 'https://example.com/story'},
      }),
    ).toEqual({
      type: 'news',
      route: 'NewsWebView',
      params: {url: 'https://example.com/story'},
    });
  });

  it('reads flat params when params object is absent', () => {
    expect(
      normalizePushNotificationData({
        type: 'debate',
        route: 'SingleDebate',
        debateId: 42,
        matchId: '1489391',
      }),
    ).toEqual({
      type: 'debate',
      route: 'SingleDebate',
      params: {debateId: 42, matchId: '1489391'},
    });
  });
});

describe('parseMatchId', () => {
  it('parses numeric and string ids', () => {
    expect(parseMatchId(99)).toBe(99);
    expect(parseMatchId('1489391')).toBe(1489391);
    expect(parseMatchId('')).toBeNull();
  });
});

describe('buildPlaceholderMatchFromId', () => {
  it('defaults to FT for post-match pushes', () => {
    const match = buildPlaceholderMatchFromId(1, {statusShort: 'FT'});
    expect(match.fixture.id).toBe(1);
    expect(match.fixture.status.short).toBe('FT');
  });
});

describe('buildMatchFromDebate', () => {
  it('uses debate teams when present', () => {
    const match = buildMatchFromDebate({
      id: 42,
      match_id: '1489391',
      headline: 'Who wins?',
      description: '',
      cards: [],
      teams: {
        home: {name: 'France', logo: 'https://home.png', score: 2},
        away: {name: 'Sweden', logo: 'https://away.png', score: 1},
      },
    });
    expect(match.fixture.id).toBe(1489391);
    expect(match.teams.home.name).toBe('France');
    expect(match.goals.home).toBe(2);
  });
});

describe('resolvePushNavigation', () => {
  it('maps news payload to NewsWebView', () => {
    const target = resolvePushNavigation({
      type: 'news',
      route: 'NewsWebView',
      params: {url: 'https://example.com/story'},
    });
    expect(target).toEqual({
      kind: 'news',
      url: 'https://example.com/story',
    });
  });

  it('falls back to Debates tab when debate lacks context', () => {
    const target = resolvePushNavigation({
      type: 'debate',
      params: {debateId: 42},
    });
    expect(target).toEqual({kind: 'debates_tab'});
  });

  it('returns null for news without url', () => {
    const target = resolvePushNavigation({type: 'news'} as PushNotificationData);
    expect(target).toBeNull();
  });

  it('maps match payload to MatchDetails when match context exists', () => {
    const match = {
      fixture: {id: 1, date: '2026-06-29T18:00:00Z', status: {long: 'FT', short: 'FT', elapsed: 0}},
      league: {id: 1, name: 'WC', logo: '', season: 2026},
      teams: {
        home: {name: 'France', logo: '', winner: null},
        away: {name: 'Sweden', logo: '', winner: null},
      },
      goals: {home: 2, away: 1},
    };
    const target = resolvePushNavigation(
      {type: 'match', params: {matchId: 1}},
      {match},
    );
    expect(target).toEqual({kind: 'match', match});
  });

  it('falls back to Home tab when match lacks context', () => {
    const target = resolvePushNavigation({type: 'match', params: {matchId: 1}});
    expect(target).toEqual({kind: 'home_tab'});
  });

  it('maps debate with context to SingleDebate', () => {
    const match = {
      fixture: {id: 1, date: '2026-06-29T18:00:00Z', status: {long: 'FT', short: 'FT', elapsed: 0}},
      league: {id: 1, name: 'WC', logo: '', season: 2026},
      teams: {
        home: {name: 'France', logo: '', winner: null},
        away: {name: 'Sweden', logo: '', winner: null},
      },
      goals: {home: null, away: null},
    };
    const debate = {
      id: 42,
      headline: 'Who wins?',
      description: '',
      cards: [],
    };
    const target = resolvePushNavigation(
      {type: 'debate', params: {debateId: 42}},
      {match, debate},
    );
    expect(target).toEqual({kind: 'debate', match, debate});
  });
});
