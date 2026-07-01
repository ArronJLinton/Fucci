import {rootNavigationRef} from './rootNavigation';
import type {PushNavigationTarget} from './pushLinking';

/** Wait briefly for NavigationContainer to mount after cold start. */
export async function waitForRootNavigationReady(
  timeoutMs = 5000,
): Promise<boolean> {
  if (rootNavigationRef.isReady()) {
    return true;
  }
  const started = Date.now();
  return new Promise(resolve => {
    const tick = () => {
      if (rootNavigationRef.isReady()) {
        resolve(true);
        return;
      }
      if (Date.now() - started >= timeoutMs) {
        resolve(false);
        return;
      }
      setTimeout(tick, 50);
    };
    tick();
  });
}

/** Navigate to the screen implied by a push notification target. */
export function navigatePushTarget(target: PushNavigationTarget): void {
  if (!rootNavigationRef.isReady()) {
    return;
  }

  const navigate = rootNavigationRef.navigate as (
    name: string,
    params?: object,
  ) => void;

  switch (target.kind) {
    case 'news':
      navigate('Main', {
        screen: 'News',
        params: {screen: 'NewsWebView', params: {url: target.url}},
      });
      break;
    case 'debate':
      navigate('Main', {
        screen: 'Debates',
        params: {
          screen: 'SingleDebate',
          params: {
            match: target.match,
            debate: target.debate,
          },
        },
      });
      break;
    case 'match':
      navigate('Main', {
        screen: 'Home',
        params: {
          screen: 'MatchDetails',
          params: {match: target.match},
        },
      });
      break;
    case 'debates_tab':
      navigate('Main', {screen: 'Debates'});
      break;
    case 'home_tab':
      navigate('Main', {screen: 'Home'});
      break;
  }
}
