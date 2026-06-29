import type {RootStackParamList} from '../types/navigation';
import type {Match} from '../types/match';
import type {DebateResponse} from '../types/debate';

export type PushNotificationData = {
  type?: 'debate' | 'match' | 'news';
  route?: keyof RootStackParamList | string;
  params?: {
    debateId?: number;
    matchId?: number | string;
    url?: string;
  };
};

export type PushNavigationTarget =
  | {screen: 'NewsWebView'; params: RootStackParamList['NewsWebView']}
  | {screen: 'SingleDebate'; params: RootStackParamList['SingleDebate']}
  | {screen: 'MatchDetails'; params: RootStackParamList['MatchDetails']}
  | {screen: 'Main'; params: RootStackParamList['Main']};

/** Maps Expo notification `data` to a root stack navigation target (Phase 1). */
export function resolvePushNavigation(
  data: PushNotificationData,
  context?: {
    debate?: DebateResponse;
    match?: Match;
  },
): PushNavigationTarget | null {
  const route = data.route ?? inferRouteFromType(data.type);
  const params = data.params ?? {};

  if (route === 'NewsWebView' || data.type === 'news') {
    const url = params.url;
    if (typeof url === 'string' && url.startsWith('http')) {
      return {screen: 'NewsWebView', params: {url}};
    }
    return null;
  }

  if (route === 'SingleDebate' || data.type === 'debate') {
    if (context?.debate && context?.match) {
      return {
        screen: 'SingleDebate',
        params: {match: context.match, debate: context.debate},
      };
    }
    return {screen: 'Main', params: {screen: 'Debates'}};
  }

  if (route === 'MatchDetails' || data.type === 'match') {
    if (context?.match) {
      return {screen: 'MatchDetails', params: {match: context.match}};
    }
    return {screen: 'Main', params: {screen: 'Home'}};
  }

  return null;
}

function inferRouteFromType(
  type?: PushNotificationData['type'],
): string | undefined {
  switch (type) {
    case 'news':
      return 'NewsWebView';
    case 'debate':
      return 'SingleDebate';
    case 'match':
      return 'MatchDetails';
    default:
      return undefined;
  }
}
