import React, {useState, useCallback} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  Switch,
  Alert,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {Ionicons} from '@expo/vector-icons';
import Constants from 'expo-constants';
import {useAuth} from '../context/AuthContext';
import {dispatchResetToMainProfileTab} from '../navigation/authNavigationActions';
import type {RootStackParamList} from '../types/navigation';

const LIME = '#c7f349';
const CYAN = '#22d3ee';
const BG = '#030712';
const CARD = '#0b1224';
const CARD_BORDER = '#1f2937';
const MUTED = '#64748b';
const TEXT = '#e2e8f0';
const ORANGE = '#f97316';

type Nav = NativeStackNavigationProp<RootStackParamList>;

const APP_NAME = 'FUCCI';

export default function SettingsScreen() {
  const navigation = useNavigation<Nav>();
  const {logout: authLogout, isLoggedIn} = useAuth();
  // const [displayMode, setDisplayMode] = useState<'dark' | 'light'>('dark');
  // const [matchAlerts, setMatchAlerts] = useState(true);
  // const [latestNews, setLatestNews] = useState(true);
  // const [socialPing, setSocialPing] = useState(false);

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
          await authLogout();
          navigation.goBack();
          dispatchResetToMainProfileTab();
        },
      },
    ]);
  }, [authLogout, navigation]);

  // const handleTerminateAccount = useCallback(() => {
  //   Alert.alert(
  //     'Terminate account?',
  //     'This would permanently remove your account and data. This action is not available in the app yet — contact support if you need to delete your account.',
  //     [{text: 'OK', style: 'default'}],
  //   );
  // }, []);

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
        {/* Account settings */}
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
            last
            onPress={() =>
              Alert.alert(
                'Privacy',
                'Your data is transmitted over HTTPS. More privacy controls coming soon.',
              )
            }
          />
        </View>

        {/* Prefs */}
        {/* <SectionHeader icon="sliders-outline" label="PREFS" />
        <View style={styles.card}>
          <Text style={styles.fieldLabel}>DISPLAY MODE</Text>
          <View style={styles.segment}>
            <TouchableOpacity
              style={[
                styles.segmentBtn,
                displayMode === 'dark' && styles.segmentBtnActive,
              ]}
              onPress={() => setDisplayMode('dark')}
              activeOpacity={0.85}>
              <Text
                style={[
                  styles.segmentText,
                  displayMode === 'dark' && styles.segmentTextActive,
                ]}>
                DARK
              </Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={[
                styles.segmentBtn,
                displayMode === 'light' && styles.segmentBtnActive,
              ]}
              onPress={() => {
                setDisplayMode('light');
                Alert.alert(
                  'Light mode',
                  'Full light theme support is coming in a future update.',
                );
              }}
              activeOpacity={0.85}>
              <Text
                style={[
                  styles.segmentText,
                  displayMode === 'light' && styles.segmentTextActive,
                ]}>
                LIGHT
              </Text>
            </TouchableOpacity>
          </View>

          <Text style={[styles.fieldLabel, styles.fieldLabelSpaced]}>
            INTERFACE LANGUAGE
          </Text>
          <TouchableOpacity
            style={styles.dropdown}
            onPress={() =>
              Alert.alert('Language', 'Additional languages coming soon.', [
                {text: 'OK'},
              ])
            }
            activeOpacity={0.85}>
            <Text style={styles.dropdownText}>English (UK)</Text>
            <Ionicons name="chevron-down" size={18} color={MUTED} />
          </TouchableOpacity>
        </View> */}

        {/* Notifications */}
        {/* <SectionHeader icon="notifications" label="NOTIFICATIONS" />
        <Text style={styles.sectionHint}>
          Control the pulse of your experience
        </Text>
        <View style={styles.card}>
          <ToggleRow
            label="MATCH ALERTS"
            value={matchAlerts}
            onValueChange={setMatchAlerts}
          />
          <ToggleRow
            label="LATEST NEWS"
            value={latestNews}
            onValueChange={setLatestNews}
          />
          <ToggleRow
            label="SOCIAL PING"
            value={socialPing}
            onValueChange={setSocialPing}
            last
          />
        </View> */}

        {/* Support */}
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

        {/* Critical zone */}
        {/* <Text style={styles.criticalTitle}>CRITICAL ZONE</Text>
        <Text style={styles.criticalCopy}>
          Deactivating your account will result in the permanent loss of all
          player stats and achievements.
        </Text> */}
        {/* <TouchableOpacity
          style={styles.terminateBtn}
          onPress={handleTerminateAccount}
          activeOpacity={0.85}>
          <Text style={styles.terminateText}>TERMINATE ACCOUNT</Text>
        </TouchableOpacity> */}

        {isLoggedIn ? (
          <TouchableOpacity
            style={styles.logoutCta}
            onPress={handleLogout}
            activeOpacity={0.9}>
            <Ionicons name="log-out-outline" size={22} color="#0f172a" />
            <Text style={styles.logoutCtaText}>LOGOUT OF SESSION</Text>
          </TouchableOpacity>
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
}: {
  icon: string;
  title: string;
  onPress: () => void;
  last?: boolean;
}) {
  return (
    <TouchableOpacity
      style={[styles.row, !last && styles.rowBorder]}
      onPress={onPress}
      activeOpacity={0.75}>
      <Ionicons name={icon as any} size={20} color={CYAN} />
      <Text style={styles.rowTitle}>{title}</Text>
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

function ToggleRow({
  label,
  value,
  onValueChange,
  last,
}: {
  label: string;
  value: boolean;
  onValueChange: (v: boolean) => void;
  last?: boolean;
}) {
  return (
    <View style={[styles.toggleRow, !last && styles.rowBorder]}>
      <Text style={styles.toggleLabel}>{label}</Text>
      <Switch
        value={value}
        onValueChange={onValueChange}
        trackColor={{false: '#334155', true: 'rgba(199,243,73,0.45)'}}
        thumbColor={value ? LIME : '#94a3b8'}
        ios_backgroundColor="#334155"
      />
    </View>
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
  heroTitle: {
    fontSize: 28,
    fontWeight: '900',
    letterSpacing: 0.5,
  },
  heroTitleLight: {
    color: TEXT,
  },
  heroTitleAccent: {
    color: LIME,
  },
  heroSub: {
    marginTop: 8,
    marginBottom: 24,
    fontSize: 11,
    fontWeight: '700',
    letterSpacing: 1.4,
    color: MUTED,
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
  sectionHint: {
    fontSize: 12,
    color: MUTED,
    marginBottom: 10,
    marginTop: -4,
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
  supportTitle: {
    flex: 1,
    fontSize: 15,
    fontWeight: '600',
    color: TEXT,
  },
  fieldLabel: {
    fontSize: 10,
    fontWeight: '800',
    letterSpacing: 1.2,
    color: MUTED,
    marginTop: 12,
    marginBottom: 8,
  },
  fieldLabelSpaced: {
    marginTop: 18,
  },
  segment: {
    flexDirection: 'row',
    borderRadius: 10,
    overflow: 'hidden',
    borderWidth: 1,
    borderColor: CARD_BORDER,
  },
  segmentBtn: {
    flex: 1,
    paddingVertical: 12,
    alignItems: 'center',
    backgroundColor: '#020617',
  },
  segmentBtnActive: {
    backgroundColor: LIME,
  },
  segmentText: {
    fontSize: 12,
    fontWeight: '900',
    letterSpacing: 1,
    color: TEXT,
  },
  segmentTextActive: {
    color: '#0f172a',
  },
  dropdown: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    backgroundColor: '#020617',
    borderWidth: 1,
    borderColor: CARD_BORDER,
    borderRadius: 10,
    paddingHorizontal: 14,
    paddingVertical: 12,
    marginBottom: 12,
  },
  dropdownText: {
    fontSize: 15,
    color: TEXT,
    fontWeight: '600',
  },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
  },
  toggleLabel: {
    fontSize: 12,
    fontWeight: '800',
    letterSpacing: 1.2,
    color: TEXT,
  },
  version: {
    fontSize: 10,
    letterSpacing: 1,
    color: MUTED,
    textAlign: 'center',
    marginBottom: 20,
  },
  criticalTitle: {
    fontSize: 13,
    fontWeight: '900',
    letterSpacing: 2,
    color: ORANGE,
    marginBottom: 8,
  },
  criticalCopy: {
    fontSize: 12,
    lineHeight: 18,
    color: MUTED,
    marginBottom: 12,
  },
  terminateBtn: {
    borderWidth: 1.5,
    borderColor: ORANGE,
    borderRadius: 10,
    paddingVertical: 14,
    alignItems: 'center',
    marginBottom: 24,
  },
  terminateText: {
    fontSize: 13,
    fontWeight: '900',
    letterSpacing: 1.5,
    color: ORANGE,
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
});
