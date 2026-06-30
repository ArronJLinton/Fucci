import {useCallback, useEffect, useState} from 'react';
import {Alert} from 'react-native';
import {useAuth} from '../context/AuthContext';
import {
  getPushPreferences,
  updatePushPreferences,
  firePushWelcomeTest,
  type PushPreferences,
} from '../services/pushApi';
import {
  registerPushWithBackend,
  unregisterPushFromBackend,
} from '../services/pushRegistration';
import {
  defaultPushPreferences,
  enabledPushPreferences,
  setPushOptedIn,
} from '../services/pushOptIn';
import {userFacingApiMessage} from '../services/api';

export function usePushPreferences() {
  const {token, isLoggedIn} = useAuth();
  const [prefs, setPrefs] = useState<PushPreferences>(defaultPushPreferences);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const loadPrefs = useCallback(async () => {
    if (!token) {
      setPrefs(defaultPushPreferences);
      return;
    }
    setLoading(true);
    try {
      const data = await getPushPreferences(token);
      setPrefs(data);
      await setPushOptedIn(data.master_enabled);
    } catch (e) {
      console.warn('[push] load preferences', e);
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    void loadPrefs();
  }, [loadPrefs]);

  const persistPrefs = useCallback(
    async (next: PushPreferences) => {
      if (!token) {
        Alert.alert('Sign in', 'Log in to manage push notifications.');
        return false;
      }
      setSaving(true);
      try {
        const saved = await updatePushPreferences(token, next);
        setPrefs(saved);
        await setPushOptedIn(saved.master_enabled);
        return true;
      } catch (e) {
        Alert.alert('Notifications', userFacingApiMessage(e));
        await loadPrefs();
        return false;
      } finally {
        setSaving(false);
      }
    },
    [token, loadPrefs],
  );

  const handleMasterToggle = useCallback(
    async (enabled: boolean) => {
      if (!token) {
        Alert.alert('Sign in', 'Log in to manage push notifications.');
        return;
      }
      if (enabled) {
        const registration = await registerPushWithBackend(token);
        if (!registration.ok) {
          Alert.alert('Notifications', registration.message);
          return;
        }
      }
      const next: PushPreferences = enabled
        ? enabledPushPreferences
        : defaultPushPreferences;
      setPrefs(next);
      const ok = await persistPrefs(next);
      if (!ok) {
        return;
      }
      if (enabled) {
        void firePushWelcomeTest(token);
        return;
      }
      await unregisterPushFromBackend(token);
    },
    [token, persistPrefs],
  );

  return {
    prefs,
    loading,
    saving,
    isLoggedIn,
    loadPrefs,
    handleMasterToggle,
  };
}
