import {useEffect, useRef} from 'react';
import {AppState, type AppStateStatus} from 'react-native';
import * as Notifications from 'expo-notifications';
import {useAuth} from '../context/AuthContext';
import {
  registerPushWithBackend,
  unregisterPushFromBackend,
} from '../services/pushRegistration';
import {
  completePendingPushOptIn,
  isPushOptedIn,
} from '../services/pushOptIn';
import {
  resolvePushNavigation,
  type PushNotificationData,
} from '../navigation/pushLinking';
import {rootNavigate} from '../navigation/rootNavigation';

/**
 * Completes pending opt-in after login, refreshes token when opted in, handles notification taps.
 */
export function usePushNotifications(): void {
  const {token, isLoggedIn, isReady} = useAuth();
  const registeredRef = useRef(false);

  useEffect(() => {
    if (!isReady || !isLoggedIn || !token) {
      registeredRef.current = false;
      return;
    }

    let cancelled = false;
    void (async () => {
      try {
        await completePendingPushOptIn(token);
        const optedIn = await isPushOptedIn();
        if (optedIn) {
          await registerPushWithBackend(token);
        }
        if (!cancelled) {
          registeredRef.current = optedIn;
        }
      } catch (e) {
        console.warn('[push] login sync failed', e);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [isReady, isLoggedIn, token]);

  useEffect(() => {
    const sub = Notifications.addNotificationResponseReceivedListener(
      (response) => {
        const raw = response.notification.request.content.data as PushNotificationData;
        const target = resolvePushNavigation(raw);
        if (!target) {
          return;
        }
        if (target.screen === 'Main') {
          rootNavigate('Main', target.params);
        } else {
          rootNavigate(target.screen, target.params);
        }
      },
    );
    return () => sub.remove();
  }, []);

  useEffect(() => {
    const onAppState = (state: AppStateStatus) => {
      if (state !== 'active' || !isLoggedIn || !token || !registeredRef.current) {
        return;
      }
      void isPushOptedIn().then((optedIn) => {
        if (optedIn) {
          registerPushWithBackend(token).catch(() => {});
        }
      });
    };
    const sub = AppState.addEventListener('change', onAppState);
    return () => sub.remove();
  }, [isLoggedIn, token]);
}

export async function logoutWithPushCleanup(
  authToken: string | null,
  logout: () => Promise<void>,
): Promise<void> {
  if (authToken) {
    try {
      await unregisterPushFromBackend(authToken);
    } catch {
      /* best effort */
    }
  }
  await logout();
}
