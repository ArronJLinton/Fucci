import {makeAuthRequest} from './api';

export type PushDevice = {
  id: number;
  expo_push_token: string;
  platform: string;
  timezone: string;
  enabled: boolean;
  last_seen_at: string;
  app_version?: string;
};

export type PushPreferences = {
  master_enabled: boolean;
  debates_enabled: boolean;
  news_enabled: boolean;
  matches_enabled: boolean;
};

export type RegisterPushDeviceRequest = {
  expo_push_token: string;
  platform: 'ios' | 'android';
  timezone: string;
  app_version?: string;
};

export async function registerPushDevice(
  token: string,
  body: RegisterPushDeviceRequest,
): Promise<PushDevice> {
  return (await makeAuthRequest(token, '/push/devices', 'POST', {
    body: JSON.stringify(body),
    headers: {'Content-Type': 'application/json'},
  })) as PushDevice;
}

export async function deletePushDevice(
  token: string,
  deviceId: number,
): Promise<void> {
  await makeAuthRequest(token, `/push/devices/${deviceId}`, 'DELETE');
}

export async function getPushPreferences(
  token: string,
): Promise<PushPreferences> {
  return (await makeAuthRequest(token, '/push/preferences', 'GET')) as PushPreferences;
}

export async function updatePushPreferences(
  token: string,
  body: Partial<PushPreferences>,
): Promise<PushPreferences> {
  return (await makeAuthRequest(token, '/push/preferences', 'PUT', {
    body: JSON.stringify(body),
    headers: {'Content-Type': 'application/json'},
  })) as PushPreferences;
}

export async function sendPushTest(token: string): Promise<void> {
  await makeAuthRequest(token, '/push/test', 'POST');
}

/** Fire-and-forget smoke test after opt-in; failures are logged, not surfaced to the user. */
export async function firePushWelcomeTest(token: string): Promise<void> {
  try {
    await sendPushTest(token);
  } catch (e) {
    console.warn('[push] welcome test notification failed:', e);
  }
}
