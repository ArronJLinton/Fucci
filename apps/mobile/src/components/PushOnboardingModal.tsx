import React, {useCallback, useEffect, useState} from 'react';
import {
  Modal,
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ActivityIndicator,
} from 'react-native';
import {Ionicons} from '@expo/vector-icons';
import {useAuth} from '../context/AuthContext';
import {
  acceptPushOnboarding,
  declinePushOnboarding,
  isPushOnboardingComplete,
} from '../services/pushOptIn';

const LIME = '#c7f349';
const BG = '#030712';
const CARD = '#0b1224';
const TEXT = '#e2e8f0';
const MUTED = '#64748b';

type Props = {
  appIsReady: boolean;
};

/**
 * Shown once on first app open to request notification permission and opt-in.
 */
export default function PushOnboardingModal({appIsReady}: Props) {
  const {token, isReady: authReady} = useAuth();
  const [visible, setVisible] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!appIsReady || !authReady) {
      return;
    }
    let cancelled = false;
    void (async () => {
      const done = await isPushOnboardingComplete();
      if (!cancelled && !done) {
        setVisible(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [appIsReady, authReady]);

  const close = useCallback(() => setVisible(false), []);

  const onEnable = useCallback(async () => {
    setBusy(true);
    try {
      await acceptPushOnboarding(token);
    } finally {
      setBusy(false);
      close();
    }
  }, [token, close]);

  const onSkip = useCallback(async () => {
    await declinePushOnboarding();
    close();
  }, [close]);

  return (
    <Modal
      visible={visible}
      animationType="fade"
      transparent
      onRequestClose={onSkip}>
      <View style={styles.backdrop}>
        <View style={styles.sheet}>
          <View style={styles.iconWrap}>
            <Ionicons name="notifications" size={28} color={LIME} />
          </View>
          <Text style={styles.title}>Never miss a moment</Text>
          <Text style={styles.body}>
            Get daily debate picks, breaking news, and match alerts during the
            World Cup. You can change this anytime in Profile or Settings.
          </Text>

          <TouchableOpacity
            style={styles.primaryBtn}
            onPress={() => void onEnable()}
            disabled={busy}
            activeOpacity={0.9}>
            {busy ? (
              <ActivityIndicator color="#0f172a" />
            ) : (
              <Text style={styles.primaryBtnText}>ENABLE NOTIFICATIONS</Text>
            )}
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.secondaryBtn}
            onPress={() => void onSkip()}
            disabled={busy}
            activeOpacity={0.75}>
            <Text style={styles.secondaryBtnText}>Not now</Text>
          </TouchableOpacity>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: 'rgba(3,7,18,0.88)',
    justifyContent: 'center',
    paddingHorizontal: 24,
  },
  sheet: {
    backgroundColor: CARD,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: 'rgba(199,243,73,0.25)',
    paddingHorizontal: 20,
    paddingVertical: 24,
  },
  iconWrap: {
    alignSelf: 'center',
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: 'rgba(199,243,73,0.12)',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 16,
  },
  title: {
    fontSize: 22,
    fontWeight: '900',
    color: TEXT,
    textAlign: 'center',
    marginBottom: 10,
  },
  body: {
    fontSize: 14,
    lineHeight: 21,
    color: MUTED,
    textAlign: 'center',
    marginBottom: 22,
  },
  primaryBtn: {
    backgroundColor: LIME,
    borderRadius: 12,
    paddingVertical: 14,
    alignItems: 'center',
    marginBottom: 10,
    minHeight: 48,
    justifyContent: 'center',
  },
  primaryBtnText: {
    fontSize: 13,
    fontWeight: '900',
    letterSpacing: 1.1,
    color: '#0f172a',
  },
  secondaryBtn: {
    paddingVertical: 10,
    alignItems: 'center',
  },
  secondaryBtnText: {
    fontSize: 14,
    fontWeight: '600',
    color: MUTED,
  },
});
