/** 007 signed-in player profile types — match API GET/POST/PUT /player-profile */

export type PlayerPosition = 'GK' | 'DEF' | 'MID' | 'FWD';

export interface PlayerProfileCareerTeam {
  id: number;
  team_name: string;
  start_year: number;
  end_year: number | null;
}

export interface PlayerProfile {
  id: number;
  age: number | null;
  country: string;
  club: string | null;
  is_free_agent: boolean;
  position: PlayerPosition;
  photo_url: string | null;
  traits: string[];
  career_teams: PlayerProfileCareerTeam[];
}

/** Local UI state before a row exists or after delete: same shape as API profile but position may be unset. */
export type PlayerProfileDraft = Omit<PlayerProfile, 'position'> & {
  position: PlayerPosition | null;
};

export type PlayerProfileOrDraft = PlayerProfile | PlayerProfileDraft;

export interface PlayerProfileInput {
  age?: number | null;
  country: string;
  club?: string | null;
  is_free_agent?: boolean;
  position: PlayerPosition;
}

/** Allowed trait codes for PUT /player-profile/traits */
export const PLAYER_TRAIT_CODES = [
  'LEADERSHIP',
  'FINESSE_SHOT',
  'PLAYMAKER',
  'SPEED_DRIBBLER',
  'LONG_SHOT_TAKER',
  'OUTSIDE_FOOT_SHOT',
  'POWER_HEADER',
  'FLAIR',
  'POWER_FREE_KICK',
] as const;

export type PlayerTraitCode = (typeof PLAYER_TRAIT_CODES)[number];
