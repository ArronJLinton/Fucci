import {Platform} from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import Constants from 'expo-constants';
import * as Device from 'expo-device';
import * as Notifications from 'expo-notifications';
import {
  deletePushDevice,
  registerPushDevice,
  type PushDevice,
} from './pushApi';
import {userFacingApiMessage} from './api';

const STORED_DEVICE_ID_KEY = 'fucci_push_device_id';

Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
    shouldShowBanner: true,
    shouldShowList: true,
  }),
});

export function getDeviceTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
  } catch {
    return 'UTC';
  }
}

async function loadStoredDeviceId(): Promise<number | null> {
  try {
    const raw = await AsyncStorage.getItem(STORED_DEVICE_ID_KEY);
    if (!raw) {
      return null;
    }
    const id = Number.parseInt(raw, 10);
    return Number.isFinite(id) ? id : null;
  } catch {
    return null;
  }
}

async function saveStoredDeviceId(id: number | null): Promise<void> {
  if (id == null) {
    await AsyncStorage.removeItem(STORED_DEVICE_ID_KEY);
    return;
  }
  await AsyncStorage.setItem(STORED_DEVICE_ID_KEY, String(id));
}

export async function requestPushPermission(): Promise<boolean> {
  if (!Device.isDevice) {
    return false;
  }
  const existing = await Notifications.getPermissionsAsync();
  if (existing.status === 'granted') {
    return true;
  }
  const requested = await Notifications.requestPermissionsAsync();
  return requested.status === 'granted';
}

export async function getExpoPushToken(): Promise<string | null> {
  if (!Device.isDevice) {
    return null;
  }
  const projectId =
    Constants.expoConfig?.extra?.eas?.projectId ??
    (Constants as {easConfig?: {projectId?: string}}).easConfig?.projectId;
  if (!projectId) {
    console.warn('[push] missing EAS projectId');
    return null;
  }
  try {
    const token = await Notifications.getExpoPushTokenAsync({projectId});
    return token.data;
  } catch (e) {
    console.error('[push] getExpoPushTokenAsync failed:', e);
    throw e;
  }
}

export type PushRegistrationFailureReason =
  | 'simulator'
  | 'permission'
  | 'project_id'
  | 'expo_token'
  | 'api';

export type PushRegistrationResult =
  | {ok: true; device: PushDevice}
  | {ok: false; reason: PushRegistrationFailureReason; message: string};

export async function registerPushWithBackend(
  authToken: string,
): Promise<PushRegistrationResult> {
  if (!Device.isDevice) {
    return {
      ok: false,
      reason: 'simulator',
      message: 'Push notifications require a physical device.',
    };
  }
  const granted = await requestPushPermission();
  if (!granted) {
    return {
      ok: false,
      reason: 'permission',
      message:
        'Notification permission was not granted. Check Settings → Fucci → Notifications.',
    };
  }
  const projectId =
    Constants.expoConfig?.extra?.eas?.projectId ??
    (Constants as {easConfig?: {projectId?: string}}).easConfig?.projectId;
  if (!projectId) {
    return {
      ok: false,
      reason: 'project_id',
      message: 'Push is not configured for this build (missing EAS project ID).',
    };
  }
  let expoToken: string;
  try {
    const token = await Notifications.getExpoPushTokenAsync({projectId});
    expoToken = token.data;
  } catch (e) {
    const detail = e instanceof Error ? e.message : String(e);
    console.error('[push] getExpoPushTokenAsync failed:', e);
    return {
      ok: false,
      reason: 'expo_token',
      message:
        detail.includes('aps-environment') || detail.includes('entitlement')
          ? 'This build is missing iOS push credentials. Upload an APNs key in Expo (eas credentials → iOS → Push Notifications).'
          : `Could not get push token: ${detail}`,
    };
  }
  try {
    const appVersion =
      Constants.expoConfig?.version ??
      (Constants as {nativeAppVersion?: string}).nativeAppVersion;
    const device = await registerPushDevice(authToken, {
      expo_push_token: expoToken,
      platform: Platform.OS === 'ios' ? 'ios' : 'android',
      timezone: getDeviceTimezone(),
      app_version: appVersion,
    });
    await saveStoredDeviceId(device.id);
    return {ok: true, device};
  } catch (e) {
    console.error('[push] registerPushDevice failed:', e);
    return {
      ok: false,
      reason: 'api',
      message: userFacingApiMessage(e),
    };
  }
}

export async function unregisterPushFromBackend(
  authToken: string,
): Promise<void> {
  const deviceId = await loadStoredDeviceId();
  if (deviceId == null) {
    return;
  }
  try {
    await deletePushDevice(authToken, deviceId);
  } catch (e) {
    console.warn('[push] unregister failed', e);
  } finally {
    await saveStoredDeviceId(null);
  }
}
