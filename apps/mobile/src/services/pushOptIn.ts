import AsyncStorage from '@react-native-async-storage/async-storage';
import {
  updatePushPreferences,
  firePushWelcomeTest,
  type PushPreferences,
} from './pushApi';
import {
  registerPushWithBackend,
  requestPushPermission,
} from './pushRegistration';

const ONBOARDING_COMPLETE_KEY = 'fucci_push_onboarding_complete';
const OPTED_IN_KEY = 'fucci_push_opted_in';
const OPT_IN_PENDING_KEY = 'fucci_push_opt_in_pending';

export const defaultPushPreferences: PushPreferences = {
  master_enabled: false,
  debates_enabled: false,
  news_enabled: false,
  matches_enabled: false,
};

export const enabledPushPreferences: PushPreferences = {
  master_enabled: true,
  debates_enabled: true,
  news_enabled: true,
  matches_enabled: true,
};

export async function isPushOnboardingComplete(): Promise<boolean> {
  return (await AsyncStorage.getItem(ONBOARDING_COMPLETE_KEY)) === '1';
}

export async function markPushOnboardingComplete(): Promise<void> {
  await AsyncStorage.setItem(ONBOARDING_COMPLETE_KEY, '1');
}

export async function isPushOptedIn(): Promise<boolean> {
  return (await AsyncStorage.getItem(OPTED_IN_KEY)) === '1';
}

export async function setPushOptedIn(optedIn: boolean): Promise<void> {
  if (optedIn) {
    await AsyncStorage.setItem(OPTED_IN_KEY, '1');
  } else {
    await AsyncStorage.removeItem(OPTED_IN_KEY);
  }
}

export async function isPushOptInPending(): Promise<boolean> {
  return (await AsyncStorage.getItem(OPT_IN_PENDING_KEY)) === '1';
}

export async function setPushOptInPending(pending: boolean): Promise<void> {
  if (pending) {
    await AsyncStorage.setItem(OPT_IN_PENDING_KEY, '1');
  } else {
    await AsyncStorage.removeItem(OPT_IN_PENDING_KEY);
  }
}

/** Request OS permission and persist opt-in; registers when auth token is available. */
export async function enablePushForUser(
  authToken: string,
  prefs: PushPreferences = enabledPushPreferences,
): Promise<boolean> {
  const granted = await requestPushPermission();
  if (!granted) {
    return false;
  }
  const device = await registerPushWithBackend(authToken);
  if (!device) {
    return false;
  }
  await updatePushPreferences(authToken, prefs);
  await setPushOptedIn(true);
  await setPushOptInPending(false);
  void firePushWelcomeTest(authToken);
  return true;
}

/**
 * First-launch flow: OS permission + local opt-in.
 * If signed in, registers device and enables prefs immediately.
 */
export async function acceptPushOnboarding(
  authToken: string | null,
): Promise<boolean> {
  await markPushOnboardingComplete();
  if (authToken) {
    return enablePushForUser(authToken);
  }
  const granted = await requestPushPermission();
  if (!granted) {
    return false;
  }
  await setPushOptInPending(true);
  return true;
}

export async function declinePushOnboarding(): Promise<void> {
  await markPushOnboardingComplete();
  await setPushOptInPending(false);
}

/** Call after login when user accepted onboarding before signing in. */
export async function completePendingPushOptIn(
  authToken: string,
): Promise<boolean> {
  const pending = await isPushOptInPending();
  const optedIn = await isPushOptedIn();
  if (!pending && !optedIn) {
    return false;
  }
  if (pending) {
    return enablePushForUser(authToken);
  }
  const device = await registerPushWithBackend(authToken);
  return device != null;
}
