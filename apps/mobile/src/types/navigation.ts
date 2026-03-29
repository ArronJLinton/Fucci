import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import type {Match} from './match';
import type {DebateResponse} from './debate';

export type MediaType = 'photo' | 'video';

/** Pending action to resume after auth (return-to-debate flow) */
export type AuthPendingAction = 'reply' | 'vote' | 'reaction' | 'swipe';

/** Params for returning to SingleDebate after login/signup */
export type ReturnToDebateParams = {
  match: Match;
  debate: DebateResponse;
  pendingAction?: AuthPendingAction;
};

/** Stack inside the Debates tab (mirrors HomeStack: list → detail) */
export type DebatesStackParamList = {
  MainDebates: undefined;
  SingleDebate: {
    match: Match;
    debate: DebateResponse;
    selectedCardIndex?: number;
    pendingAction?: AuthPendingAction;
  };
  NewsWebView: {
    url: string;
  };
};

/** Tab screens inside the Main (bottom tab) navigator */
export type MainTabParamList = {
  Home: undefined;
  News: undefined;
  Debates: undefined;
  Profile: undefined;
};

/** Root stack screens (Main = tab navigator, SignUp, Login, CameraPreview) and nested screen names for typing navigate() */
export type RootStackParamList = {
  Main: undefined | {screen?: keyof MainTabParamList};
  SignUp: undefined | {returnToDebate?: ReturnToDebateParams};
  Login: undefined | {returnToDebate?: ReturnToDebateParams};
  ForgotPassword: undefined;
  Settings: undefined;
  HomeTab: undefined;
  MatchDetails: {
    match: Match;
  };
  SingleDebate: {
    match: Match;
    debate: DebateResponse;
    selectedCardIndex?: number;
    /** Set when returning from Login/SignUp to resume action (best-effort) */
    pendingAction?: AuthPendingAction;
  };
  Table: {
    match: Match;
  };
  NewsFeed: undefined;
  NewsWebView: {
    url: string;
  };
  CameraPreview: {
    onPhotoCapture: (uri: string, type: MediaType) => void;
  };
  CreatePlayerProfile: undefined;
  PlayerProfile: undefined;
};

export type NavigationProp = NativeStackNavigationProp<RootStackParamList>;
