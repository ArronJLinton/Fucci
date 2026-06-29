import type {NavigationState} from '@react-navigation/native';
import * as ScreenOrientation from 'expo-screen-orientation';

/** Route that may rotate (news + match YouTube Shorts). */
export const ORIENTATION_UNLOCK_ROUTE = 'MatchTeamShorts';

export function lockAppPortrait(): void {
  ScreenOrientation.lockAsync(
    ScreenOrientation.OrientationLock.PORTRAIT_UP,
  ).catch(() => {});
}

export function unlockAppOrientation(): void {
  ScreenOrientation.unlockAsync().catch(() => {});
}

export function getActiveRouteName(
  state: NavigationState | undefined,
): string | undefined {
  if (!state) {
    return undefined;
  }
  const route = state.routes[state.index];
  const nested = route.state as NavigationState | undefined;
  if (nested) {
    return getActiveRouteName(nested);
  }
  return route.name;
}

export function isOrientationUnlockRoute(
  state: NavigationState | undefined,
): boolean {
  return getActiveRouteName(state) === ORIENTATION_UNLOCK_ROUTE;
}
