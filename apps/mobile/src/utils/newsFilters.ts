import type {NewsArticle} from '../types/news';
import type {League} from '../constants/leagues';

export type NewsCategoryId = 'all' | 'transfers' | 'match' | 'injury';

export function mergeAndSortArticles(
  today: NewsArticle[],
  history: NewsArticle[],
): NewsArticle[] {
  const map = new Map<string, NewsArticle>();
  for (const a of [...today, ...history]) {
    if (!map.has(a.id)) {
      map.set(a.id, a);
    }
  }
  return [...map.values()].sort(
    (a, b) =>
      new Date(b.publishedAt).getTime() - new Date(a.publishedAt).getTime(),
  );
}

function articleText(a: NewsArticle): string {
  return `${a.title} ${a.snippet || ''}`.toLowerCase();
}

export function filterByCategory(
  articles: NewsArticle[],
  category: NewsCategoryId,
): NewsArticle[] {
  if (category === 'all') {
    return articles;
  }
  const keys: Record<Exclude<NewsCategoryId, 'all'>, string[]> = {
    transfers: [
      'transfer',
      'signing',
      'sign ',
      'deal',
      'contract',
      'fee',
      'loan',
      'bid',
      'move to',
      'joins',
      'arrival',
    ],
    match: [
      'match',
      'full time',
      'full-time',
      'highlight',
      'score',
      ' beat ',
      'won ',
      'defeat',
      'draw',
      'victory',
      'goal',
    ],
    injury: [
      'injury',
      'injured',
      'sideline',
      'out for',
      'hamstring',
      'knock',
      'surgery',
      'weeks out',
      'ruled out',
    ],
  };
  const needles = keys[category];
  return articles.filter(a => needles.some(n => articleText(a).includes(n)));
}

const LEAGUE_KEYWORDS: Record<number, string[]> = {
  39: ['premier league', 'epl', 'english premier', ' england '],
  140: ['la liga', 'laliga', ' spain ', 'spanish'],
  135: ['serie a', ' italy ', 'italian'],
  78: ['bundesliga', ' germany ', 'german'],
  61: ['ligue 1', ' ligue  ', ' france ', 'french'],
  2: ['champions league', ' ucl ', 'european'],
  0: [
    'world cup',
    'international',
    'fifa',
    'nations league',
    'euro 20',
    'copa america',
    'africa cup',
  ],
};

export function filterByLeague(
  articles: NewsArticle[],
  league: League | null,
): NewsArticle[] {
  if (league == null) {
    return articles;
  }
  const extra = [league.name.toLowerCase()];
  const keywords = [...(LEAGUE_KEYWORDS[league.id] ?? []), ...extra];
  return articles.filter(a =>
    keywords.some(k => articleText(a).includes(k)),
  );
}

/** Small uppercase tag for the card (lime) */
export function articleCategoryLabel(article: NewsArticle): string {
  const t = articleText(article);
  if (
    t.includes('injury') ||
    t.includes('injured') ||
    t.includes('sideline') ||
    t.includes('ruled out')
  ) {
    return 'INJURY UPDATE';
  }
  if (
    t.includes('transfer') ||
    t.includes('signing') ||
    t.includes('contract') ||
    t.includes(' loan ')
  ) {
    return 'TRANSFER TALK';
  }
  if (
    t.includes('match') ||
    t.includes(' beat ') ||
    t.includes('won ') ||
    t.includes('score') ||
    t.includes('goal')
  ) {
    return 'MATCH REPORT';
  }
  return 'FOOTBALL';
}
