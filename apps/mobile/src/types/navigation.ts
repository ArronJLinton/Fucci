import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import type {Match} from './match';
import type {DebateResponse} from './debate';

export type MediaType = 'photo' | 'video';

/** Tab screens inside the Main (bottom tab) navigator */
export type MainTabParamList = {
  Home: undefined;
  News: undefined;
  Profile: undefined;
};

/** Root stack screens (Main = tab navigator, SignUp, Login, CameraPreview) and nested screen names for typing navigate() */
export type RootStackParamList = {
  Main: undefined | {screen?: keyof MainTabParamList};
  SignUp: undefined;
  Login: undefined;
  HomeTab: undefined;
  MatchDetails: {
    match: Match;
  };
  SingleDebate: {
    match: Match;
    debate: DebateResponse;
    selectedCardIndex?: number;
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
};

export type NavigationProp = NativeStackNavigationProp<RootStackParamList>;
