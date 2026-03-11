import React, {useState, useEffect, useCallback} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
  TextInput,
  ActivityIndicator,
  Alert,
  Image,
} from 'react-native';
import {useNavigation, useRoute} from '@react-navigation/native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {Ionicons} from '@expo/vector-icons';
import {useAuth} from '../context/AuthContext';
import {rootNavigate} from '../navigation/rootNavigation';
import {
  getProfile,
  updateProfile,
  getFollowing,
  type AuthUser,
  type FollowingItem,
} from '../services/api';

type TabId = 'following' | 'profile' | 'team';

interface SettingsScreenProps {
  /** When true, screen is embedded in Profile tab (no back button) */
  embeddedInTab?: boolean;
}

type SettingsRouteParams = {embeddedInTab?: boolean};

export default function SettingsScreen({
  embeddedInTab: embeddedInTabProp,
}: SettingsScreenProps = {}) {
  const navigation = useNavigation();
  const route = useRoute();
  const embeddedInTab =
    embeddedInTabProp ??
    (route.params as SettingsRouteParams | undefined)?.embeddedInTab ??
    false;
  const {token, isLoggedIn, logout: authLogout} = useAuth();
  const [profile, setProfile] = useState<AuthUser | null>(null);
  const [following, setFollowing] = useState<FollowingItem[]>([]);
  const [activeTab, setActiveTab] = useState<TabId>('following');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [profileEdit, setProfileEdit] = useState({
    firstname: '',
    lastname: '',
    avatar_url: '',
  });

  const loadProfileData = useCallback(async (t: string) => {
    const [userData, followData] = await Promise.all([
      getProfile(t),
      getFollowing(t),
    ]);
    if (userData) {
      setProfile(userData);
      setProfileEdit({
        firstname: userData.firstname || '',
        lastname: userData.lastname || '',
        avatar_url: userData.avatar_url || '',
      });
    }
    setFollowing(followData || []);
    setLoading(false);
  }, []);

  useEffect(() => {
    if (!token) {
      setLoading(false);
      return;
    }
    loadProfileData(token);
  }, [token, loadProfileData]);

  const handleBack = () => {
    if (navigation.canGoBack()) {
      navigation.goBack();
    } else {
      rootNavigate('Main');
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
          rootNavigate('Login');
        },
      },
    ]);
  };

  const handleSaveProfile = async () => {
    if (!token) return;
    setSaving(true);
    const updated = await updateProfile(token, {
      firstname: profileEdit.firstname || undefined,
      lastname: profileEdit.lastname || undefined,
      avatar_url: profileEdit.avatar_url || undefined,
    });
    setSaving(false);
    if (updated) {
      setProfile(updated);
    }
  };

  if (loading && token) {
    const body = (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color="#007AFF" />
      </View>
    );
    return embeddedInTab ? (
      body
    ) : (
      <SafeAreaView style={styles.container} edges={['top']}>
        {body}
      </SafeAreaView>
    );
  }

  const displayName = profile
    ? [profile.firstname, profile.lastname].filter(Boolean).join(' ') ||
      profile.email
    : '';

  const mainContent = (
    <>
      {!embeddedInTab && (
        <View style={styles.header}>
          <TouchableOpacity onPress={handleBack} style={styles.backButton}>
            <Ionicons name="chevron-back" size={28} color="#007AFF" />
          </TouchableOpacity>
          <Text style={styles.headerTitle}>Settings</Text>
        </View>
      )}

      <View style={styles.profileSummary}>
        {isLoggedIn && profile?.avatar_url ? (
          <Image source={{uri: profile.avatar_url}} style={styles.avatar} />
        ) : (
          <View style={styles.avatarPlaceholder}>
            <Ionicons name="person" size={40} color="#999" />
          </View>
        )}
        <View style={styles.profileInfo}>
          <Text style={styles.profileName}>
            {isLoggedIn ? displayName : 'Account'}
          </Text>
          <Text style={styles.profileEmail}>
            {isLoggedIn ? profile?.email : 'Login or register to continue'}
          </Text>
        </View>
      </View>

      <View style={styles.tabs}>
        {(['following', 'profile', 'team'] as const).map(tab => (
          <TouchableOpacity
            key={tab}
            style={[styles.tab, activeTab === tab && styles.tabActive]}
            onPress={() => setActiveTab(tab)}>
            <Text
              style={[
                styles.tabText,
                activeTab === tab && styles.tabTextActive,
              ]}>
              {tab === 'following'
                ? 'Following'
                : tab === 'profile'
                  ? 'Player Profile'
                  : 'Team Manager'}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      <ScrollView
        style={styles.tabContent}
        showsVerticalScrollIndicator={false}>
        {!isLoggedIn ? (
          <View style={styles.loginPromptContainer}>
            <Text style={styles.loginPromptText}>
              <Text style={styles.loginLinkText} onPress={() => rootNavigate('Login')}>Login</Text>
              <Text> or </Text>
              <Text style={styles.loginLinkText} onPress={() => rootNavigate('SignUp')}>Register</Text>
              <Text> to access account features.</Text>
            </Text>
          </View>
        ) : (
          <>
            {activeTab === 'following' && (
              <View style={styles.section}>
                {following.length === 0 ? (
                  <Text style={styles.emptyText}>
                    You're not following any leagues or teams yet.
                  </Text>
                ) : (
                  following.map(item => (
                    <View key={item.id} style={styles.followRow}>
                      <Text style={styles.followLabel}>
                        {item.type}: {item.name || item.followable_id}
                      </Text>
                      <Text style={styles.followHint}>
                        Follow toggle (coming soon)
                      </Text>
                    </View>
                  ))
                )}
              </View>
            )}

            {activeTab === 'profile' && (
              <View style={styles.section}>
                <Text style={styles.label}>First name</Text>
                <TextInput
                  style={styles.input}
                  value={profileEdit.firstname}
                  onChangeText={t =>
                    setProfileEdit(p => ({...p, firstname: t}))
                  }
                  placeholder="First name"
                  placeholderTextColor="#999"
                />
                <Text style={styles.label}>Last name</Text>
                <TextInput
                  style={styles.input}
                  value={profileEdit.lastname}
                  onChangeText={t => setProfileEdit(p => ({...p, lastname: t}))}
                  placeholder="Last name"
                  placeholderTextColor="#999"
                />
                <Text style={styles.label}>Avatar URL</Text>
                <TextInput
                  style={styles.input}
                  value={profileEdit.avatar_url}
                  onChangeText={t =>
                    setProfileEdit(p => ({...p, avatar_url: t}))
                  }
                  placeholder="https://..."
                  placeholderTextColor="#999"
                  autoCapitalize="none"
                />
                <TouchableOpacity
                  style={[styles.button, saving && styles.buttonDisabled]}
                  onPress={handleSaveProfile}
                  disabled={saving}>
                  {saving ? (
                    <ActivityIndicator color="#fff" />
                  ) : (
                    <Text style={styles.buttonText}>Save</Text>
                  )}
                </TouchableOpacity>
              </View>
            )}

            {activeTab === 'team' && (
              <View style={styles.section}>
                {profile?.role === 'team_manager' ? (
                  <Text style={styles.emptyText}>
                    Team management content (coming soon).
                  </Text>
                ) : (
                  <Text style={styles.emptyText}>
                    Request access to become a team manager.
                  </Text>
                )}
              </View>
            )}
          </>
        )}
      </ScrollView>

      {isLoggedIn && (
        <TouchableOpacity style={styles.logoutButton} onPress={handleLogout}>
          <Ionicons name="log-out-outline" size={22} color="#c00" />
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
    backgroundColor: '#fff',
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#fff',
    padding: 24,
  },
  placeholderText: {
    fontSize: 16,
    color: '#666',
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
    color: '#666',
    textAlign: 'center',
    width: '100%',
  },
  loginLinkText: {
    fontSize: 16,
    color: '#007AFF',
    textDecorationLine: 'underline',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 8,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#e0e0e0',
  },
  backButton: {
    padding: 8,
  },
  headerTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#000',
    marginLeft: 8,
  },
  profileSummary: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#e0e0e0',
  },
  avatar: {
    width: 56,
    height: 56,
    borderRadius: 28,
  },
  avatarPlaceholder: {
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: '#eee',
    justifyContent: 'center',
    alignItems: 'center',
  },
  profileInfo: {
    flex: 1,
    marginLeft: 16,
  },
  profileName: {
    fontSize: 17,
    fontWeight: '600',
    color: '#000',
  },
  profileEmail: {
    fontSize: 14,
    color: '#666',
    marginTop: 2,
  },
  tabs: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: '#e0e0e0',
  },
  tab: {
    flex: 1,
    paddingVertical: 12,
    alignItems: 'center',
  },
  tabActive: {
    borderBottomWidth: 2,
    borderBottomColor: '#007AFF',
  },
  tabText: {
    fontSize: 14,
    color: '#666',
  },
  tabTextActive: {
    color: '#007AFF',
    fontWeight: '600',
  },
  tabContent: {
    flex: 1,
    padding: 16,
  },
  section: {
    marginBottom: 24,
  },
  label: {
    fontSize: 14,
    fontWeight: '500',
    color: '#333',
    marginBottom: 6,
  },
  input: {
    borderWidth: 1,
    borderColor: '#ddd',
    borderRadius: 10,
    paddingHorizontal: 14,
    paddingVertical: 12,
    fontSize: 16,
    marginBottom: 16,
    color: '#000',
  },
  emptyText: {
    fontSize: 15,
    color: '#666',
    textAlign: 'center',
    marginTop: 24,
  },
  followRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  followLabel: {
    fontSize: 15,
    color: '#000',
  },
  followHint: {
    fontSize: 12,
    color: '#999',
  },
  button: {
    backgroundColor: '#007AFF',
    borderRadius: 10,
    paddingVertical: 14,
    alignItems: 'center',
    marginTop: 8,
  },
  buttonDisabled: {
    opacity: 0.7,
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  linkButton: {
    marginTop: 12,
    paddingVertical: 8,
    alignItems: 'center',
  },
  linkText: {
    color: '#007AFF',
    fontSize: 16,
  },
  logoutButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 16,
    borderTopWidth: 1,
    borderTopColor: '#e0e0e0',
    gap: 8,
  },
  logoutText: {
    fontSize: 16,
    color: '#c00',
    fontWeight: '600',
  },
});
