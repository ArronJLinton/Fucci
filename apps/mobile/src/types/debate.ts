export interface DebateStance {
  id: string;
  type: 'agree' | 'disagree' | 'wildcard';
  text: string;
  votes: number;
}

export interface DebateCard {
  id?: number;
  debate_id?: number;
  stance: 'agree' | 'disagree' | 'wildcard';
  title: string;
  description: string;
  vote_counts?: { upvotes: number; downvotes: number; emojis?: Record<string, number> };
}

export interface DebateContent {
  headline: string;
  description: string;
  cards: DebateCard[];
}

export interface DebatePrompt extends DebateContent {}

/** Analytics snippet on feed summaries (009 GET …/public-feed, …/feed) */
export interface DebateAnalyticsSummary {
  total_votes: number;
  total_comments: number;
  engagement_score: number;
}

/**
 * Minimal debate row for main feeds (009) — aligns with contracts/debates-feed.yaml.
 * Optional provenance fields when API/DB expose them (004).
 */
export interface DebateSummary {
  id: number;
  match_id: string;
  headline: string;
  description?: string;
  debate_type: string;
  ai_generated?: boolean;
  created_at: string;
  updated_at?: string;
  analytics?: DebateAnalyticsSummary;
  /** Authenticated “voted” list — latest swipe time for sorting */
  last_voted_at?: string;
  source_headline?: string;
  source_url?: string;
  source_published_at?: string;
}

/** Guest browse — GET /debates/public-feed */
export interface PublicDebateFeedResponse {
  debates: DebateSummary[];
}

/** Signed-in — GET /debates/feed */
export interface DebateFeedResponse {
  new_debates: DebateSummary[];
  voted_debates: DebateSummary[];
}

/** List item from GET /debates/match (no cards) */
export interface DebateListItem {
  id: number;
  match_id: string;
  debate_type: string;
  headline: string;
  description: string;
  ai_generated: boolean;
  created_at: string;
  updated_at: string;
}

/** Card vote totals for live meter (006 swipe voting) */
export interface CardVoteTotals {
  total_yes: number;
  total_no: number;
}

/** Response from PUT /debates/:id/cards/:cardId/vote */
export interface CardVoteCounts {
  card_id: number;
  yes_count: number;
  no_count: number;
  total_yes?: number;
  total_no?: number;
}

/** Full debate with cards (GET /debates/:id or POST /debates/generate response) */
export interface DebateResponse extends DebateContent {
  id?: number;
  match_id?: string;
  debate_type?: string;
  ai_generated?: boolean;
  created_at?: string;
  updated_at?: string;
  cards: DebateCard[];
  card_vote_totals?: CardVoteTotals;
}

export type DebateType = 'pre_match' | 'post_match';

/** Mock comment for T031 (no API yet) */
export interface MockComment {
  id: string;
  username: string;
  content: string;
  upvotes: number;
  replies?: number;
}

/** Reaction count per emoji (006 GET /debates/:id/comments) */
export interface ReactionCount {
  emoji: string;
  count: number;
}

/** Comment from GET /debates/:id/comments — top-level and subcomments, no stance/seeded in API */
export interface DebateComment {
  id: number;
  debate_id: number;
  parent_comment_id?: number | null;
  user_id: number;
  user_display_name: string;
  user_avatar_url?: string | null;
  content: string;
  created_at: string;
  net_score: number;
  current_user_vote?: 'upvote' | 'downvote' | null;
  reactions: ReactionCount[];
  subcomments?: DebateComment[];
}
