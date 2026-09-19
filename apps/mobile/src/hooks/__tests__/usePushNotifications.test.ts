/// <reference types="jest" />

import * as Notifications from 'expo-notifications';
import {consumePushNotificationResponse} from '../usePushNotifications';
import {
  navigatePushTarget,
  waitForRootNavigationReady,
} from '../../navigation/navigatePushTarget';
import {prefetchPushContext} from '../../navigation/prefetchPushContext';
import {
  normalizePushNotificationData,
  resolvePushNavigation,
} from '../../navigation/pushLinking';

jest.mock('expo-notifications', () => ({
  clearLastNotificationResponseAsync: jest.fn(),
  addNotificationResponseReceivedListener: jest.fn(),
  getLastNotificationResponseAsync: jest.fn(),
}));
jest.mock('../../context/AuthContext', () => ({useAuth: jest.fn()}));
jest.mock('../../config/queryClient', () => ({queryClient: {}}));
jest.mock('../../services/pushRegistration', () => ({
  registerPushWithBackend: jest.fn(),
  unregisterPushFromBackend: jest.fn(),
}));
jest.mock('../../services/pushOptIn', () => ({
  completePendingPushOptIn: jest.fn(),
  isPushOptedIn: jest.fn(),
}));
jest.mock('../../navigation/navigatePushTarget', () => ({
  navigatePushTarget: jest.fn(),
  waitForRootNavigationReady: jest.fn(),
}));
jest.mock('../../navigation/prefetchPushContext', () => ({
  prefetchPushContext: jest.fn(),
}));
jest.mock('../../navigation/pushLinking', () => ({
  normalizePushNotificationData: jest.fn(),
  resolvePushNavigation: jest.fn(),
}));

const response = {
  notification: {
    request: {
      identifier: 'notification-1',
      content: {data: {type: 'news', url: 'https://example.com/news'}},
    },
  },
} as unknown as Notifications.NotificationResponse;

describe('consumePushNotificationResponse', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.mocked(waitForRootNavigationReady).mockResolvedValue(true);
    jest.mocked(normalizePushNotificationData).mockReturnValue({
      type: 'news',
      params: {url: 'https://example.com/news'},
    });
    jest.mocked(prefetchPushContext).mockResolvedValue({});
    jest.mocked(resolvePushNavigation).mockReturnValue({
      kind: 'news',
      url: 'https://example.com/news',
    });
    jest
      .mocked(Notifications.clearLastNotificationResponseAsync)
      .mockResolvedValue(undefined);
  });

  it('clears the last response after navigating', async () => {
    await consumePushNotificationResponse(response, 'auth-token');

    expect(navigatePushTarget).toHaveBeenCalledWith({
      kind: 'news',
      url: 'https://example.com/news',
    });
    expect(
      Notifications.clearLastNotificationResponseAsync,
    ).toHaveBeenCalledTimes(1);
  });

  it('clears the last response when deep-link handling fails', async () => {
    jest.mocked(prefetchPushContext).mockRejectedValue(new Error('network error'));
    const warn = jest.spyOn(console, 'warn').mockImplementation(() => {});

    await consumePushNotificationResponse(response, 'auth-token');

    expect(navigatePushTarget).not.toHaveBeenCalled();
    expect(
      Notifications.clearLastNotificationResponseAsync,
    ).toHaveBeenCalledTimes(1);
    warn.mockRestore();
  });
});
