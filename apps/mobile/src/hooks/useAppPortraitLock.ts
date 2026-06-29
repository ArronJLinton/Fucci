import {useEffect} from 'react';
import {AppState, type AppStateStatus} from 'react-native';
import type {NavigationState} from '@react-navigation/native';
import {
  isOrientationUnlockRoute,
  lockAppPortrait,
} from '../utils/screenOrientation';

/**
 * Keeps the app portrait-locked everywhere except MatchTeamShorts (news/match stories).
 */
export function useAppPortraitLock(
  navigationState: NavigationState | undefined,
  enabled: boolean,
): void {
  const onShortsScreen = isOrientationUnlockRoute(navigationState);

  useEffect(() => {
    if (!enabled || onShortsScreen) {
      return;
    }
    lockAppPortrait();
  }, [enabled, onShortsScreen]);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    const sub = AppState.addEventListener('change', (next: AppStateStatus) => {
      if (next === 'active' && !isOrientationUnlockRoute(navigationState)) {
        lockAppPortrait();
      }
    });
    return () => sub.remove();
  }, [enabled, navigationState]);
}
