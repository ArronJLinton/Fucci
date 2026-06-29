/// <reference types="jest" />

import AsyncStorage from '@react-native-async-storage/async-storage';
import {
  acceptPushOnboarding,
  declinePushOnboarding,
  defaultPushPreferences,
  enabledPushPreferences,
  isPushOnboardingComplete,
  markPushOnboardingComplete,
} from '../pushOptIn';
import * as pushApi from '../pushApi';
import * as pushRegistration from '../pushRegistration';

jest.mock('../pushApi');
jest.mock('../pushRegistration');

describe('pushOptIn', () => {
  beforeEach(async () => {
    await AsyncStorage.clear();
    jest.clearAllMocks();
  });

  it('enabledPushPreferences turns on master and all categories', () => {
    expect(enabledPushPreferences).toEqual({
      master_enabled: true,
      debates_enabled: true,
      news_enabled: true,
      matches_enabled: true,
    });
    expect(defaultPushPreferences.master_enabled).toBe(false);
  });

  it('declinePushOnboarding marks complete without opt-in', async () => {
    await declinePushOnboarding();
    expect(await isPushOnboardingComplete()).toBe(true);
  });

  it('acceptPushOnboarding with auth token registers and enables all prefs', async () => {
    jest.spyOn(pushRegistration, 'requestPushPermission').mockResolvedValue(true);
    jest.spyOn(pushRegistration, 'registerPushWithBackend').mockResolvedValue({
      id: 1,
      expo_push_token: 'ExponentPushToken[abc]',
      platform: 'ios',
      timezone: 'UTC',
      enabled: true,
      last_seen_at: new Date().toISOString(),
    });
    jest.spyOn(pushApi, 'updatePushPreferences').mockResolvedValue(enabledPushPreferences);
    jest.spyOn(pushApi, 'firePushWelcomeTest').mockResolvedValue(undefined);

    const ok = await acceptPushOnboarding('token-123');
    expect(ok).toBe(true);
    expect(pushApi.updatePushPreferences).toHaveBeenCalledWith(
      'token-123',
      enabledPushPreferences,
    );
    expect(pushApi.firePushWelcomeTest).toHaveBeenCalledWith('token-123');
    expect(await isPushOnboardingComplete()).toBe(true);
  });

  it('acceptPushOnboarding without auth sets pending when permission granted', async () => {
    jest.spyOn(pushRegistration, 'requestPushPermission').mockResolvedValue(true);

    const ok = await acceptPushOnboarding(null);
    expect(ok).toBe(true);
    expect(pushApi.updatePushPreferences).not.toHaveBeenCalled();
    expect(await isPushOnboardingComplete()).toBe(true);
  });

  it('markPushOnboardingComplete persists flag', async () => {
    expect(await isPushOnboardingComplete()).toBe(false);
    await markPushOnboardingComplete();
    expect(await isPushOnboardingComplete()).toBe(true);
  });
});
