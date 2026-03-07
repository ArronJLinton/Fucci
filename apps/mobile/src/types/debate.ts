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

/** Full debate with cards (GET /debates/:id or POST /debates/generate response) */
export interface DebateResponse extends DebateContent {
  id?: number;
  match_id?: string;
  debate_type?: string;
  ai_generated?: boolean;
  created_at?: string;
  updated_at?: string;
  cards: DebateCard[];
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
