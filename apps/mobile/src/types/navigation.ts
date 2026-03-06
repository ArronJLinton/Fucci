import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import type {Match} from './match';
import type {DebateResponse} from './debate';

export type MediaType = 'photo' | 'video';

export type RootStackParamList = {
  Main: undefined;
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
