// Shared TypeScript types for Fucci API
// Generated from OpenAPI specification

export interface User {
  id: string;
  email: string;
  role: 'fan' | 'team_manager' | 'admin';
  display_name: string;
  avatar_url?: string;
  is_verified: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
}

export interface Team {
  id: string;
  name: string;
  short_name?: string;
  logo_url?: string;
  banner_url?: string;
  team_type: 'professional' | 'community';
  league_id?: string;
  founded_year?: number;
  home_venue?: string;
  description?: string;
  is_verified: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Player {
  id: string;
  team_id: string;
  first_name: string;
  last_name: string;
  position?: string;
  jersey_number?: number;
  date_of_birth?: string;
  nationality?: string;
  height_cm?: number;
  weight_kg?: number;
  photo_url?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Match {
  id: string;
  home_team_id: string;
  away_team_id: string;
  league_id?: string;
  match_date: string;
  venue?: string;
  status: 'scheduled' | 'live' | 'finished' | 'postponed' | 'cancelled';
  home_score: number;
  away_score: number;
  match_minute: number;
  referee?: string;
  attendance?: number;
  weather_conditions?: string;
  created_at: string;
  updated_at: string;
}

export interface MatchEvent {
  id: string;
  match_id: string;
  event_type: 'goal' | 'yellow_card' | 'red_card' | 'substitution' | 'penalty';
  minute: number;
  player_id?: string;
  team_id: string;
  description?: string;
  created_at: string;
}

export interface Debate {
  id: string;
  match_id?: string;
  title: string;
  description: string;
  side_a_title: string;
  side_b_title: string;
  side_a_description: string;
  side_b_description: string;
  ai_generated: boolean;
  generation_prompt?: string;
  bias_score?: number;
  quality_score?: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface DebateResponse {
  id: string;
  debate_id: string;
  user_id: string;
  content: string;
  side_chosen?: 'side_a' | 'side_b';
  upvotes: number;
  downvotes: number;
  is_approved: boolean;
  created_at: string;
  updated_at: string;
}

export interface Story {
  id: string;
  user_id: string;
  match_id?: string;
  team_id?: string;
  content_type: 'photo' | 'video' | 'text';
  media_url: string;
  caption?: string;
  expires_at: string;
  view_count: number;
  is_active: boolean;
  created_at: string;
}

export interface Report {
  id: string;
  reporter_id: string;
  reportable_type: 'debate' | 'debate_response' | 'story' | 'user';
  reportable_id: string;
  reason:
    | 'spam'
    | 'harassment'
    | 'inappropriate_content'
    | 'fake_team'
    | 'other';
  description?: string;
  status: 'pending' | 'reviewed' | 'resolved' | 'dismissed';
  admin_notes?: string;
  created_at: string;
  updated_at: string;
}

// API Response types
export interface ApiResponse<T> {
  data: T;
  message?: string;
  success: boolean;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}

export interface ApiError {
  error: string;
  message: string;
  details?: Record<string, any>;
}

// Request types
export interface CreateUserRequest {
  email: string;
  password: string;
  display_name: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface UpdateUserRequest {
  display_name?: string;
  avatar_url?: string;
}

export interface FollowRequest {
  followable_type: 'team' | 'player' | 'match';
  followable_id: string;
}

export interface CreateTeamRequest {
  name: string;
  short_name?: string;
  team_type: 'professional' | 'community';
  league_id?: string;
  founded_year?: number;
  home_venue?: string;
  description?: string;
}

export interface UpdateTeamRequest {
  name?: string;
  short_name?: string;
  founded_year?: number;
  home_venue?: string;
  description?: string;
}

export interface CreatePlayerRequest {
  first_name: string;
  last_name: string;
  position?: string;
  jersey_number?: number;
  date_of_birth?: string;
  nationality?: string;
  height_cm?: number;
  weight_kg?: number;
  photo_url?: string;
}

export interface CreateDebateResponseRequest {
  content: string;
  side_chosen?: 'side_a' | 'side_b';
}

export interface VoteRequest {
  vote_type: 'upvote' | 'downvote';
}

export interface CreateStoryRequest {
  match_id?: string;
  team_id?: string;
  content_type: 'photo' | 'video' | 'text';
  media_url: string;
  caption?: string;
}

export interface CreateReportRequest {
  reportable_type: 'debate' | 'debate_response' | 'story' | 'user';
  reportable_id: string;
  reason:
    | 'spam'
    | 'harassment'
    | 'inappropriate_content'
    | 'fake_team'
    | 'other';
  description?: string;
}

// Query parameters
export interface ListTeamsParams {
  type?: 'professional' | 'community';
  league_id?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export interface ListMatchesParams {
  status?: 'scheduled' | 'live' | 'finished' | 'postponed' | 'cancelled';
  team_id?: string;
  date_from?: string;
  date_to?: string;
  limit?: number;
  offset?: number;
}

export interface ListStoriesParams {
  match_id?: string;
  team_id?: string;
  limit?: number;
  offset?: number;
}

export interface ListDebateResponsesParams {
  limit?: number;
  offset?: number;
}

export interface ListReportsParams {
  status?: 'pending' | 'reviewed' | 'resolved' | 'dismissed';
  limit?: number;
  offset?: number;
}

// Authentication types
export interface LoginResponse {
  user: User;
  token: string;
}

export interface AuthContext {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<LoginResponse>;
  logout: () => void;
  register: (userData: CreateUserRequest) => Promise<User>;
}

// WebSocket types
export interface WebSocketMessage {
  type: 'match_update' | 'debate_notification' | 'story_notification' | 'error';
  data: any;
  timestamp: string;
}

export interface MatchUpdateMessage {
  type: 'match_update';
  data: {
    match_id: string;
    home_score: number;
    away_score: number;
    match_minute: number;
    status: string;
    events?: MatchEvent[];
  };
  timestamp: string;
}

export interface DebateNotificationMessage {
  type: 'debate_notification';
  data: {
    debate_id: string;
    match_id?: string;
    title: string;
    user_id: string;
  };
  timestamp: string;
}

export interface StoryNotificationMessage {
  type: 'story_notification';
  data: {
    story_id: string;
    user_id: string;
    content_type: string;
    match_id?: string;
    team_id?: string;
  };
  timestamp: string;
}

// Cache types
export interface CacheConfig {
  match_ttl: number; // 5 days in seconds
  debate_ttl: number; // 1 hour in seconds
  user_ttl: number; // 24 hours in seconds
  team_ttl: number; // 24 hours in seconds
}

// Environment types
export interface EnvironmentConfig {
  API_BASE_URL: string;
  WS_URL: string;
  APP_NAME: string;
  APP_VERSION: string;
  NODE_ENV: 'development' | 'production' | 'test';
  DEBUG: boolean;
}

// Navigation types
export interface RootStackParamList {
  Home: undefined;
  MatchDetails: { matchId: string };
  TeamDetails: { teamId: string };
  PlayerDetails: { playerId: string };
  DebateDetails: { debateId: string };
  StoryDetails: { storyId: string };
  Profile: undefined;
  Account: undefined;
  Settings: undefined;
  Login: undefined;
  Register: undefined;
}

export interface TabParamList {
  Home: undefined;
  Matches: undefined;
  Debates: undefined;
  Stories: undefined;
  Profile: undefined;
}

// Utility types
export type Status = 'idle' | 'loading' | 'success' | 'error';

export interface AsyncState<T> {
  data: T | null;
  status: Status;
  error: string | null;
}

export type SortOrder = 'asc' | 'desc';

export interface SortConfig {
  field: string;
  order: SortOrder;
}

// Form types
export interface FormField {
  name: string;
  label: string;
  type:
    | 'text'
    | 'email'
    | 'password'
    | 'number'
    | 'date'
    | 'select'
    | 'textarea';
  required: boolean;
  placeholder?: string;
  options?: { label: string; value: string }[];
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    message?: string;
  };
}

export interface FormState {
  values: Record<string, any>;
  errors: Record<string, string>;
  touched: Record<string, boolean>;
  isSubmitting: boolean;
  isValid: boolean;
}
