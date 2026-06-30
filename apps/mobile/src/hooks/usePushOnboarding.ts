import {useEffect} from 'react';
import {useAuth} from '../context/AuthContext';
import {
  acceptPushOnboarding,
  isPushOnboardingComplete,
} from '../services/pushOptIn';

/**
 * On first app open, request notification permission via the native OS dialog only.
 */
export function usePushOnboarding(appIsReady: boolean): void {
  const {token, isReady: authReady} = useAuth();

  useEffect(() => {
    if (!appIsReady || !authReady) {
      return;
    }
    let cancelled = false;
    void (async () => {
      const done = await isPushOnboardingComplete();
      if (cancelled || done) {
        return;
      }
      await acceptPushOnboarding(token);
    })();
    return () => {
      cancelled = true;
    };
  }, [appIsReady, authReady, token]);
}
