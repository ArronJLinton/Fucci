import {useEffect, useRef} from 'react';
import {AppState, type AppStateStatus} from 'react-native';
import * as Notifications from 'expo-notifications';
import {useAuth} from '../context/AuthContext';
import {queryClient} from '../config/queryClient';
import {
  registerPushWithBackend,
  unregisterPushFromBackend,
} from '../services/pushRegistration';
import {
  completePendingPushOptIn,
  isPushOptedIn,
} from '../services/pushOptIn';
import {normalizePushNotificationData, resolvePushNavigation} from '../navigation/pushLinking';
import {prefetchPushContext} from '../navigation/prefetchPushContext';
import {
  navigatePushTarget,
  waitForRootNavigationReady,
} from '../navigation/navigatePushTarget';

/**
 * Completes pending opt-in after login, refreshes token when opted in, handles notification taps.
 */
export function usePushNotifications(): void {
  const {token, isLoggedIn, isReady} = useAuth();
  const registeredRef = useRef(false);
  const handledNotificationIdRef = useRef<string | null>(null);
  const handlingRef = useRef(false);

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
    const handleResponse = (response: Notifications.NotificationResponse) => {
      const notificationId = response.notification.request.identifier;
      if (handledNotificationIdRef.current === notificationId || handlingRef.current) {
        return;
      }
      handlingRef.current = true;
      handledNotificationIdRef.current = notificationId;

      void (async () => {
        try {
          await waitForRootNavigationReady();
          const raw = response.notification.request.content.data;
          const data = normalizePushNotificationData(
            raw && typeof raw === 'object'
              ? (raw as Record<string, unknown>)
              : {},
          );
          const context = await prefetchPushContext(data, {
            token,
            queryClient,
          });
          const target = resolvePushNavigation(data, context);
          if (target) {
            navigatePushTarget(target);
          }
        } catch (e) {
          console.warn('[push] deep link failed', e);
        } finally {
          handlingRef.current = false;
        }
      })();
    };

    const sub = Notifications.addNotificationResponseReceivedListener(handleResponse);

    void Notifications.getLastNotificationResponseAsync().then(response => {
      if (response) {
        handleResponse(response);
      }
    });

    return () => sub.remove();
  }, [token]);

  useEffect(() => {
    const onAppState = (state: AppStateStatus) => {
      if (state !== 'active' || !isLoggedIn || !token || !registeredRef.current) {
        return;
      }
      void isPushOptedIn().then(optedIn => {
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
