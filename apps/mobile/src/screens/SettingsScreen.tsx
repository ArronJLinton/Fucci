import React, {useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
  Alert,
  ActivityIndicator,
} from 'react-native';
import {useRoute} from '@react-navigation/native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {Ionicons} from '@expo/vector-icons';
import {useAuth} from '../context/AuthContext';
import {rootNavigate, rootResetTo} from '../navigation/rootNavigation';
import {getPlayerProfile} from '../services/playerProfile';

interface SettingsScreenProps {
  /** When true, screen is embedded in Profile tab (no back button) */
  embeddedInTab?: boolean;
}

type SettingsRouteParams = {embeddedInTab?: boolean};

export default function SettingsScreen({
  embeddedInTab: embeddedInTabProp,
}: SettingsScreenProps = {}) {
  const route = useRoute();
  const embeddedInTab =
    embeddedInTabProp ??
    (route.params as SettingsRouteParams | undefined)?.embeddedInTab ??
    false;
  const {user, isLoggedIn, logout: authLogout, token} = useAuth();
  const [playerModeLoading, setPlayerModeLoading] = useState(false);

  const handleOpenPlayerMode = async () => {
    if (!token) return;
    setPlayerModeLoading(true);
    try {
      const profile = await getPlayerProfile(token);
      if (profile) {
        rootNavigate('PlayerProfile');
      } else {
        rootNavigate('CreatePlayerProfile');
      }
    } catch {
      Alert.alert(
        'Something went wrong',
        'We could not check your player profile. Check your connection and try again.',
      );
    } finally {
      setPlayerModeLoading(false);
    }
  };

  const handleLogout = () => {
    Alert.alert('Log out?', 'Are you sure you want to log out?', [
      {text: 'Cancel', style: 'cancel'},
      {
        text: 'Log out',
        style: 'destructive',
        onPress: async () => {
          await authLogout();
          rootResetTo('Login');
        },
      },
    ]);
  };

  const mainContent = (
    <>
      <View style={styles.headerRightRow}>
        <View />
        <TouchableOpacity style={styles.headerGear}>
          <Ionicons name="settings" size={18} color="#c7f349" />
        </TouchableOpacity>
      </View>

      <ScrollView
        style={styles.tabContent}
        showsVerticalScrollIndicator={false}>
        {!isLoggedIn ? (
          <View style={styles.loginPromptContainer}>
            <Text style={styles.loginPromptText}>
              <Text
                style={styles.loginLinkText}
                onPress={() => rootNavigate('Login')}>
                Login
              </Text>
              <Text> or </Text>
              <Text
                style={styles.loginLinkText}
                onPress={() => rootNavigate('SignUp')}>
                Register
              </Text>
              <Text> to access account features.</Text>
            </Text>
          </View>
        ) : (
          <>
            <View style={styles.section}>
              <Text style={styles.accountHeading}>Account Settings</Text>

              <TouchableOpacity style={styles.settingsCard}>
                <View style={styles.settingsCardIconWrap}>
                  <Ionicons name="people" size={16} color="#d9f99d" />
                </View>
                <View style={styles.settingsCardTextWrap}>
                  <Text style={styles.settingsCardTitle}>Following</Text>
                  <Text style={styles.settingsCardSub}>
                    View teams and leagues you follow
                  </Text>
                </View>
                <Ionicons name="chevron-forward" size={18} color="#94a3b8" />
              </TouchableOpacity>

              <TouchableOpacity
                style={styles.settingsCard}
                onPress={handleOpenPlayerMode}
                disabled={playerModeLoading}
                accessibilityLabel="Open Player Mode"
                accessibilityState={{busy: playerModeLoading}}>
                <View style={styles.settingsCardIconWrap}>
                  <Ionicons name="person" size={16} color="#d9f99d" />
                </View>
                <View style={styles.settingsCardTextWrap}>
                  <Text style={styles.settingsCardTitle}>Player Mode</Text>
                  <Text style={styles.settingsCardSub}>
                    Open player profile portal
                  </Text>
                </View>
                {playerModeLoading ? (
                  <ActivityIndicator size="small" color="#94a3b8" />
                ) : (
                  <Ionicons name="chevron-forward" size={18} color="#94a3b8" />
                )}
              </TouchableOpacity>

              <TouchableOpacity style={styles.settingsCard}>
                <View style={styles.settingsCardIconWrap}>
                  <Ionicons name="briefcase" size={16} color="#d9f99d" />
                </View>
                <View style={styles.settingsCardTextWrap}>
                  <Text style={styles.settingsCardTitle}>Team Manager</Text>
                  <Text style={styles.settingsCardSub}>
                    {user?.role === 'team_manager'
                      ? 'Manage your team tools'
                      : 'Request team manager access'}
                  </Text>
                </View>
                <Ionicons name="chevron-forward" size={18} color="#94a3b8" />
              </TouchableOpacity>
            </View>
          </>
        )}
      </ScrollView>

      {isLoggedIn && (
        <TouchableOpacity style={styles.logoutButton} onPress={handleLogout}>
          <Ionicons name="log-out-outline" size={18} color="#f87171" />
          <Text style={styles.logoutText}>Log out</Text>
        </TouchableOpacity>
      )}
    </>
  );

  return embeddedInTab ? (
    <View style={styles.container}>{mainContent}</View>
  ) : (
    <SafeAreaView style={styles.container} edges={['top']}>
      {mainContent}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#030712',
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#030712',
    padding: 24,
  },
  placeholderText: {
    fontSize: 16,
    color: '#94a3b8',
    marginBottom: 16,
  },
  loginPromptContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingVertical: 48,
    paddingHorizontal: 24,
  },
  loginPromptText: {
    fontSize: 16,
    color: '#94a3b8',
    textAlign: 'center',
    width: '100%',
  },
  loginLinkText: {
    fontSize: 16,
    color: '#67e8f9',
    textDecorationLine: 'underline',
  },
  headerRightRow: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingTop: 10,
    paddingBottom: 4,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#111827',
    backgroundColor: '#07101f',
  },
  backButton: {
    padding: 8,
  },
  headerAvatarDot: {
    width: 26,
    height: 26,
    borderRadius: 13,
    borderWidth: 1.5,
    borderColor: '#c7f349',
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
  },
  headerAvatarImg: {width: 24, height: 24, borderRadius: 12},
  headerGear: {
    width: 28,
    height: 28,
    borderRadius: 14,
    alignItems: 'center',
    justifyContent: 'center',
  },
  headerTitle: {
    flex: 1,
    marginLeft: 10,
    fontSize: 30,
    fontWeight: '900',
    letterSpacing: 1.4,
    color: '#c7f349',
    fontStyle: 'italic',
  },
  profileSummary: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingTop: 14,
    paddingBottom: 14,
    paddingHorizontal: 20,
  },
  profileAvatarWrap: {
    width: 84,
    height: 84,
    borderRadius: 14,
    borderWidth: 2,
    borderColor: '#84cc16',
    backgroundColor: '#111827',
    alignItems: 'center',
    justifyContent: 'center',
    position: 'relative',
    overflow: 'hidden',
  },
  avatar: {
    width: 76,
    height: 76,
    borderRadius: 10,
  },
  avatarPlaceholder: {
    width: 76,
    height: 76,
    borderRadius: 10,
    backgroundColor: '#0b1224',
    justifyContent: 'center',
    alignItems: 'center',
  },
  avatarEditBtn: {
    width: 36,
    height: 36,
    borderRadius: 8,
    backgroundColor: '#111827',
    borderWidth: 1,
    borderColor: '#1f2937',
    alignItems: 'center',
    justifyContent: 'center',
    marginLeft: 12,
  },
  profileInfo: {
    flex: 1,
    alignItems: 'flex-start',
    marginTop: 0,
    paddingHorizontal: 14,
  },
  profileName: {
    fontSize: 45,
    fontWeight: '900',
    fontStyle: 'italic',
    color: '#e2e8f0',
    textTransform: 'uppercase',
    textAlign: 'left',
  },
  profileEmail: {
    fontSize: 11,
    color: '#94a3b8',
    marginTop: 4,
    letterSpacing: 1.2,
    textTransform: 'uppercase',
  },
  tabs: {
    flexDirection: 'row',
    borderWidth: 1,
    borderColor: '#1f2937',
    borderRadius: 10,
    marginHorizontal: 20,
    marginBottom: 6,
    overflow: 'hidden',
    backgroundColor: '#111827',
  },
  tab: {
    flex: 1,
    paddingVertical: 14,
    alignItems: 'center',
  },
  tabActive: {
    backgroundColor: '#1f2937',
  },
  tabText: {
    fontSize: 12,
    color: '#94a3b8',
    letterSpacing: 0.8,
    textTransform: 'uppercase',
  },
  tabTextActive: {
    color: '#e2e8f0',
    fontWeight: '700',
  },
  tabContent: {
    flex: 1,
    padding: 16,
  },
  errorBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: '#2c1f17',
    borderBottomWidth: 1,
    borderBottomColor: '#7c2d12',
  },
  errorBannerText: {
    flex: 1,
    fontSize: 14,
    color: '#fdba74',
  },
  retryButton: {
    paddingVertical: 6,
    paddingHorizontal: 12,
  },
  retryButtonText: {
    fontSize: 14,
    color: '#67e8f9',
    fontWeight: '600',
  },
  saveErrorText: {
    fontSize: 14,
    color: '#f87171',
    marginBottom: 12,
  },
  section: {
    marginBottom: 20,
  },
  accountHeading: {
    fontSize: 12,
    color: '#94a3b8',
    fontWeight: '700',
    letterSpacing: 2,
    textTransform: 'uppercase',
    marginBottom: 12,
  },
  settingsCard: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#0b1224',
    borderWidth: 1,
    borderColor: '#1f2937',
    borderRadius: 12,
    paddingVertical: 14,
    paddingHorizontal: 12,
    marginBottom: 12,
  },
  settingsCardIconWrap: {
    width: 40,
    height: 40,
    borderRadius: 8,
    backgroundColor: '#111827',
    alignItems: 'center',
    justifyContent: 'center',
  },
  settingsCardTextWrap: {
    flex: 1,
    marginLeft: 12,
  },
  settingsCardTitle: {
    fontSize: 22,
    fontWeight: '800',
    color: '#e2e8f0',
  },
  settingsCardSub: {
    marginTop: 2,
    fontSize: 13,
    color: '#94a3b8',
  },
  personalInfoEditor: {
    borderWidth: 1,
    borderColor: '#1f2937',
    borderRadius: 12,
    backgroundColor: '#0a1220',
    padding: 12,
    marginBottom: 12,
  },
  playerProfileEntry: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 12,
    paddingHorizontal: 10,
    marginBottom: 16,
    borderWidth: 1,
    borderColor: '#1f2937',
    borderRadius: 10,
    backgroundColor: '#0b1224',
  },
  playerProfileEntryText: {
    flex: 1,
    marginLeft: 12,
    fontSize: 16,
    color: '#d1d5db',
  },
  label: {
    fontSize: 11,
    fontWeight: '700',
    color: '#94a3b8',
    marginBottom: 8,
    letterSpacing: 1.2,
    textTransform: 'uppercase',
  },
  input: {
    borderWidth: 1,
    borderColor: '#1f2937',
    borderRadius: 8,
    paddingHorizontal: 14,
    paddingVertical: 12,
    fontSize: 20,
    marginBottom: 16,
    color: '#f8fafc',
    backgroundColor: '#020617',
  },
  emptyText: {
    fontSize: 15,
    color: '#94a3b8',
    textAlign: 'center',
    marginTop: 24,
  },
  followRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#1f2937',
  },
  followLabel: {
    fontSize: 15,
    color: '#d1d5db',
  },
  followHint: {
    fontSize: 12,
    color: '#64748b',
  },
  button: {
    backgroundColor: '#c7f349',
    borderRadius: 12,
    paddingVertical: 18,
    alignItems: 'center',
    marginTop: 8,
    shadowColor: '#bef264',
    shadowOffset: {width: 0, height: 8},
    shadowOpacity: 0.35,
    shadowRadius: 16,
    elevation: 8,
  },
  buttonDisabled: {
    opacity: 0.7,
  },
  buttonText: {
    color: '#0f172a',
    fontSize: 20,
    fontWeight: '900',
    letterSpacing: 1.2,
    textTransform: 'uppercase',
    fontStyle: 'italic',
  },
  linkButton: {
    marginTop: 12,
    paddingVertical: 8,
    alignItems: 'center',
  },
  linkText: {
    color: '#67e8f9',
    fontSize: 16,
  },
  logoutButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 16,
    borderWidth: 1,
    borderColor: '#3f1d1d',
    borderRadius: 10,
    marginHorizontal: 120,
    marginBottom: 8,
    gap: 8,
  },
  logoutText: {
    fontSize: 15,
    color: '#f87171',
    fontWeight: '700',
    letterSpacing: 1.1,
    textTransform: 'uppercase',
  },
  buildFooter: {
    textAlign: 'center',
    color: '#475569',
    fontSize: 10,
    letterSpacing: 0.8,
    marginTop: 8,
    marginBottom: 14,
  },
  stadiumBand: {
    height: 90,
    marginHorizontal: 0,
    borderBottomWidth: 1,
    borderBottomColor: '#1e293b',
    backgroundColor: '#0a1220',
  },
});
