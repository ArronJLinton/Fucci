import {
  createNavigationContainerRef,
  CommonActions,
} from '@react-navigation/native';
import type {RootStackParamList} from '../types/navigation';

export const rootNavigationRef =
  createNavigationContainerRef<RootStackParamList>();

export function rootNavigate(
  name: keyof RootStackParamList,
  params?: RootStackParamList[keyof RootStackParamList],
) {
  if (rootNavigationRef.isReady()) {
    rootNavigationRef.navigate(name as any, params as any);
  }
}

/**
 * Resets the root stack to a single screen (e.g. Login). Use after logout so
 * Back cannot return to authenticated screens.
 */
export function rootResetTo(
  name: keyof RootStackParamList,
  params?: RootStackParamList[keyof RootStackParamList],
) {
  if (rootNavigationRef.isReady()) {
    rootNavigationRef.dispatch(
      CommonActions.reset({
        index: 0,
        routes: [{name, params: params as any}],
      }),
    );
  }
}
