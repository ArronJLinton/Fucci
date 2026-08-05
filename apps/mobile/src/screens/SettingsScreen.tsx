import React, {useCallback, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  Alert,
  ActivityIndicator,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {Ionicons} from '@expo/vector-icons';
import Constants from 'expo-constants';
import {useAuth} from '../context/AuthContext';
import {dispatchResetToMainProfileTab} from '../navigation/authNavigationActions';
import type {RootStackParamList} from '../types/navigation';
import PushNotificationSettings from '../components/PushNotificationSettings';
import {logoutWithPushCleanup} from '../hooks/usePushNotifications';
import {deleteAccount, userFacingApiMessage} from '../services/api';

const LIME = '#c7f349';
const CYAN = '#22d3ee';
const BG = '#030712';
const CARD = '#0b1224';
const CARD_BORDER = '#1f2937';
const MUTED = '#64748b';
const TEXT = '#e2e8f0';
const DANGER = '#f87171';

type Nav = NativeStackNavigationProp<RootStackParamList>;

const APP_NAME = 'FUCCI';

export default function SettingsScreen() {
  const navigation = useNavigation<Nav>();
  const {logout: authLogout, isLoggedIn, token} = useAuth();
  const [isDeletingAccount, setIsDeletingAccount] = useState(false);

  const appVersion =
    Constants.expoConfig?.version ??
    (Constants as {nativeAppVersion?: string}).nativeAppVersion ??
    '1.0.0';

  const handleLogout = useCallback(() => {
    Alert.alert('Log out?', 'Are you sure you want to log out?', [
      {text: 'Cancel', style: 'cancel'},
      {
        text: 'Log out',
        style: 'destructive',
        onPress: async () => {
          await logoutWithPushCleanup(token, authLogout);
          navigation.goBack();
          dispatchResetToMainProfileTab();
        },
      },
    ]);
  }, [authLogout, navigation, token]);

  const performDeleteAccount = useCallback(async () => {
    if (!token || isDeletingAccount) {
      return;
    }
    setIsDeletingAccount(true);
    try {
      await deleteAccount(token);
      await logoutWithPushCleanup(null, authLogout);
      navigation.goBack();
      dispatchResetToMainProfileTab();
    } catch (error) {
      Alert.alert('Could not delete account', userFacingApiMessage(error));
    } finally {
      setIsDeletingAccount(false);
    }
  }, [authLogout, isDeletingAccount, navigation, token]);

  const handleDeleteAccount = useCallback(() => {
    if (!token || isDeletingAccount) {
      return;
    }
    Alert.alert(
      'Delete account?',
      'This permanently deletes your account and associated data. This cannot be undone.',
      [
        {text: 'Cancel', style: 'cancel'},
        {
          text: 'Continue',
          style: 'destructive',
          onPress: () => {
            Alert.alert(
              'Confirm deletion',
              'Are you sure you want to permanently delete your Fucci account?',
              [
                {text: 'Cancel', style: 'cancel'},
                {
                  text: 'Delete Account',
                  style: 'destructive',
                  onPress: () => {
                    void performDeleteAccount();
                  },
                },
              ],
            );
          },
        },
      ],
    );
  }, [isDeletingAccount, performDeleteAccount, token]);

  return (
    <SafeAreaView style={styles.safe} edges={['top']}>
      <View style={styles.topBar}>
        <View style={styles.topBarSide}>
          <TouchableOpacity
            onPress={() => navigation.goBack()}
            hitSlop={{top: 12, bottom: 12, left: 12, right: 12}}
            accessibilityRole="button"
            accessibilityLabel="Go back">
            <View style={styles.topBarIconBox}>
              <Ionicons name="person" size={18} color={LIME} />
            </View>
          </TouchableOpacity>
        </View>
        <View style={styles.topBarCenter}>
          <Text style={styles.brandMark}>{APP_NAME}</Text>
        </View>
        <View style={styles.topBarSide} />
      </View>

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}>
        <SectionHeader icon="person" label="ACCOUNT SETTINGS" />
        <View style={styles.card}>
          <SettingsRow
            icon="person-outline"
            title="Profile Identity"
            onPress={() =>
              Alert.alert(
                'Profile Identity',
                'Edit your name and avatar from the Account tab.',
              )
            }
          />
          <SettingsRow
            icon="lock-closed-outline"
            title="Security & Password"
            onPress={() =>
              Alert.alert(
                'Security & Password',
                'Password changes will be available in a future update.',
              )
            }
          />
          <SettingsRow
            icon="shield-checkmark-outline"
            title="Privacy Encryption"
            last={!isLoggedIn}
            onPress={() =>
              Alert.alert(
                'Privacy',
                'Your data is transmitted over HTTPS. More privacy controls coming soon.',
              )
            }
          />
          {isLoggedIn ? (
            <SettingsRow
              icon="trash-outline"
              title="Delete Account"
              danger
              last
              onPress={handleDeleteAccount}
              disabled={isDeletingAccount}
            />
          ) : null}
        </View>

        <PushNotificationSettings showHeader />

        <SectionHeader icon="help-circle" label="SUPPORT" />
        <View style={styles.card}>
          <SupportRow
            icon="book-outline"
            title="FAQ"
            onPress={() => Alert.alert('FAQ', 'Help center is coming soon.')}
          />
          <SupportRow
            icon="headset-outline"
            title="Contact Support"
            onPress={() =>
              Alert.alert(
                'Contact',
                'Support contact options will be available in a future update.',
              )
            }
          />
          <SupportRow
            icon="document-text-outline"
            title="Terms of Service"
            last
            onPress={() =>
              Alert.alert('Terms', 'Terms of service will be published here.')
            }
          />
        </View>
        <Text style={styles.version}>VERSION {appVersion}-FUCCI_BETA</Text>

        {isLoggedIn ? (
          <TouchableOpacity
            style={styles.logoutCta}
            onPress={handleLogout}
            activeOpacity={0.9}
            disabled={isDeletingAccount}>
            <Ionicons name="log-out-outline" size={22} color="#0f172a" />
            <Text style={styles.logoutCtaText}>LOGOUT OF SESSION</Text>
          </TouchableOpacity>
        ) : null}

        {isDeletingAccount ? (
          <View style={styles.deletingRow}>
            <ActivityIndicator color={DANGER} />
            <Text style={styles.deletingText}>Deleting account…</Text>
          </View>
        ) : null}

        <View style={{height: 32}} />
      </ScrollView>
    </SafeAreaView>
  );
}

function SectionHeader({icon, label}: {icon: string; label: string}) {
  return (
    <View style={styles.sectionHeaderRow}>
      <Ionicons name={icon as any} size={16} color={LIME} />
      <Text style={styles.sectionHeaderText}>{label}</Text>
    </View>
  );
}

function SettingsRow({
  icon,
  title,
  onPress,
  last,
  danger,
  disabled,
}: {
  icon: string;
  title: string;
  onPress: () => void;
  last?: boolean;
  danger?: boolean;
  disabled?: boolean;
}) {
  const tint = danger ? DANGER : CYAN;
  return (
    <TouchableOpacity
      style={[styles.row, !last && styles.rowBorder, disabled && styles.rowDisabled]}
      onPress={onPress}
      activeOpacity={0.75}
      disabled={disabled}
      accessibilityRole="button"
      accessibilityLabel={title}>
      <Ionicons name={icon as any} size={20} color={tint} />
      <Text style={[styles.rowTitle, danger && styles.rowTitleDanger]}>{title}</Text>
      <Ionicons name="chevron-forward" size={18} color={MUTED} />
    </TouchableOpacity>
  );
}

function SupportRow({
  icon,
  title,
  onPress,
  last,
}: {
  icon: string;
  title: string;
  onPress: () => void;
  last?: boolean;
}) {
  return (
    <TouchableOpacity
      style={[styles.supportRow, !last && styles.rowBorder]}
      onPress={onPress}
      activeOpacity={0.75}>
      <Ionicons name={icon as any} size={20} color={CYAN} />
      <Text style={styles.supportTitle}>{title}</Text>
      <Ionicons name="chevron-forward" size={18} color={MUTED} />
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
    backgroundColor: BG,
  },
  topBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: 'rgba(255,255,255,0.08)',
  },
  topBarSide: {
    width: 40,
    alignItems: 'flex-start',
    justifyContent: 'center',
  },
  topBarCenter: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  topBarIconBox: {
    width: 36,
    height: 36,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: 'rgba(199,243,73,0.35)',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: 'rgba(15,23,42,0.6)',
  },
  brandMark: {
    fontSize: 18,
    fontWeight: '900',
    fontStyle: 'italic',
    letterSpacing: 1.2,
    color: LIME,
  },
  scroll: {
    flex: 1,
  },
  scrollContent: {
    paddingHorizontal: 16,
    paddingTop: 20,
  },
  sectionHeaderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 10,
    marginTop: 8,
  },
  sectionHeaderText: {
    fontSize: 12,
    fontWeight: '800',
    letterSpacing: 1.6,
    color: LIME,
  },
  card: {
    backgroundColor: CARD,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: CARD_BORDER,
    paddingHorizontal: 14,
    paddingVertical: 4,
    marginBottom: 20,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 14,
    gap: 12,
  },
  supportRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 14,
    gap: 12,
  },
  rowBorder: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: 'rgba(148,163,184,0.15)',
  },
  rowTitle: {
    flex: 1,
    fontSize: 15,
    fontWeight: '700',
    color: TEXT,
  },
  rowTitleDanger: {
    color: DANGER,
  },
  rowDisabled: {
    opacity: 0.5,
  },
  supportTitle: {
    flex: 1,
    fontSize: 15,
    fontWeight: '600',
    color: TEXT,
  },
  version: {
    fontSize: 10,
    letterSpacing: 1,
    color: MUTED,
    textAlign: 'center',
    marginBottom: 20,
  },
  logoutCta: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 10,
    backgroundColor: LIME,
    borderRadius: 12,
    paddingVertical: 16,
    marginBottom: 8,
  },
  logoutCtaText: {
    fontSize: 14,
    fontWeight: '900',
    letterSpacing: 1.2,
    color: '#0f172a',
  },
  deletingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 10,
    marginTop: 12,
    marginBottom: 8,
  },
  deletingText: {
    fontSize: 13,
    fontWeight: '600',
    color: DANGER,
  },
});
