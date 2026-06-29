import {
  resolvePushNavigation,
  type PushNotificationData,
} from '../pushLinking';

describe('resolvePushNavigation', () => {
  it('maps news payload to NewsWebView', () => {
    const target = resolvePushNavigation({
      type: 'news',
      route: 'NewsWebView',
      params: {url: 'https://example.com/story'},
    });
    expect(target).toEqual({
      screen: 'NewsWebView',
      params: {url: 'https://example.com/story'},
    });
  });

  it('falls back to Debates tab when debate lacks context', () => {
    const target = resolvePushNavigation({
      type: 'debate',
      params: {debateId: 42},
    });
    expect(target).toEqual({
      screen: 'Main',
      params: {screen: 'Debates'},
    });
  });

  it('returns null for news without url', () => {
    const target = resolvePushNavigation({type: 'news'} as PushNotificationData);
    expect(target).toBeNull();
  });

  it('maps match payload to MatchDetails when match context exists', () => {
    const match = {
      fixture: {id: 1, date: '2026-06-29T18:00:00Z', status: {long: 'NS', short: 'NS', elapsed: 0}},
      league: {id: 1, name: 'WC', logo: '', season: 2026},
      teams: {
        home: {name: 'France', logo: '', winner: null},
        away: {name: 'Sweden', logo: '', winner: null},
      },
      goals: {home: null, away: null},
    };
    const target = resolvePushNavigation(
      {type: 'match', params: {matchId: 1}},
      {match},
    );
    expect(target).toEqual({screen: 'MatchDetails', params: {match}});
  });

  it('falls back to Home tab when match lacks context', () => {
    const target = resolvePushNavigation({type: 'match', params: {matchId: 1}});
    expect(target).toEqual({
      screen: 'Main',
      params: {screen: 'Home'},
    });
  });

  it('maps debate with context to SingleDebate', () => {
    const match = {
      fixture: {id: 1, date: '2026-06-29T18:00:00Z', status: {long: 'NS', short: 'NS', elapsed: 0}},
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
    expect(target).toEqual({screen: 'SingleDebate', params: {match, debate}});
  });
});
