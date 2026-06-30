import {useCallback} from 'react';
import {useAuth} from '../context/AuthContext';
import {
  acceptPushOnboarding,
  isPushOnboardingComplete,
} from '../services/pushOptIn';

/**
 * Returns a `requestOnboarding` callback that callers should invoke from an
 * explicit user action (e.g. an onboarding sheet button or a settings toggle).
 * It will no-op if onboarding has already been completed.
 */
export function usePushOnboarding(appIsReady: boolean): {
  requestOnboarding: () => Promise<void>;
} {
  const {token, isReady: authReady} = useAuth();

  const requestOnboarding = useCallback(async () => {
    if (!appIsReady || !authReady) {
      return;
    }
    const done = await isPushOnboardingComplete();
    if (done) {
      return;
    }
    await acceptPushOnboarding(token);
  }, [appIsReady, authReady, token]);

  return {requestOnboarding};
}
