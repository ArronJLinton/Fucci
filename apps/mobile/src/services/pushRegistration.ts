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
  const token = await Notifications.getExpoPushTokenAsync({projectId});
  return token.data;
}

export async function registerPushWithBackend(
  authToken: string,
): Promise<PushDevice | null> {
  const granted = await requestPushPermission();
  if (!granted) {
    return null;
  }
  const expoToken = await getExpoPushToken();
  if (!expoToken) {
    return null;
  }
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
  return device;
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
