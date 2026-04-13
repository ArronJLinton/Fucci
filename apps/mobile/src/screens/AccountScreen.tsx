import React, {useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
  Alert,
  ActivityIndicator,
  Image,
} from 'react-native';
import * as ImagePicker from 'expo-image-picker';
import {useRoute} from '@react-navigation/native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {Ionicons} from '@expo/vector-icons';
import {useAuth} from '../context/AuthContext';
import {rootNavigate} from '../navigation/rootNavigation';
import {getPlayerProfile} from '../services/playerProfile';
import {updateProfile} from '../services/auth';
import {ApiRequestError, userFacingApiMessage} from '../services/api';
import {uploadToCloudinary} from '../services/cloudinaryUpload';
import ProfileGuestAuth from '../components/ProfileGuestAuth';
import {dispatchResetToMainProfileTab} from '../navigation/authNavigationActions';
import type {ReturnToDebateParams} from '../types/navigation';

interface AccountScreenProps {
  /** When true, screen is embedded in Profile tab (no back button) */
  embeddedInTab?: boolean;
}

type AccountRouteParams = {
  embeddedInTab?: boolean;
  returnToDebate?: ReturnToDebateParams;
};

export default function AccountScreen({
  embeddedInTab: embeddedInTabProp,
}: AccountScreenProps = {}) {
  const route = useRoute();
  const routeParams = (route.params as AccountRouteParams | undefined) ?? {};
  const embeddedInTab = embeddedInTabProp ?? routeParams.embeddedInTab ?? false;
  const profileAuthReturnToDebate = routeParams.returnToDebate;
  const {user, isLoggedIn, logout: authLogout, token, setAuth} = useAuth();
  const [playerModeLoading, setPlayerModeLoading] = useState(false);
  const [avatarUploading, setAvatarUploading] = useState(false);
  const [avatarURL, setAvatarURL] = useState<string | null>(
    user?.avatar_url ?? null,
  );

  React.useEffect(() => {
    setAvatarURL(user?.avatar_url ?? null);
  }, [user?.avatar_url]);

  const handlePickAndUploadAvatar = async (source: 'camera' | 'library') => {
    if (!token || !user) return;
    try {
      const permission =
        source === 'camera'
          ? await ImagePicker.requestCameraPermissionsAsync()
          : await ImagePicker.requestMediaLibraryPermissionsAsync();
      if (!permission.granted) {
        Alert.alert(
          'Permission needed',
          `Please allow ${source === 'camera' ? 'camera' : 'photo library'} access in Settings to update your avatar.`,
        );
        return;
      }

      const result =
        source === 'camera'
          ? await ImagePicker.launchCameraAsync({
              mediaTypes: ImagePicker.MediaTypeOptions.Images,
              allowsEditing: false,
              quality: 0.8,
            })
          : await ImagePicker.launchImageLibraryAsync({
              mediaTypes: ImagePicker.MediaTypeOptions.Images,
              allowsEditing: false,
              quality: 0.8,
            });

      if (result.canceled || result.assets.length === 0) return;
      const asset = result.assets[0];

      setAvatarUploading(true);
      const secureURL = await uploadToCloudinary(token, 'avatar', {
        uri: asset.uri,
        fileName: asset.fileName ?? undefined,
        mimeType: asset.mimeType ?? undefined,
        size: asset.fileSize ?? undefined,
      });
      const updatedUser = await updateProfile(token, {avatar_url: secureURL});
      if (!updatedUser) {
        throw new Error('Could not save your profile. Please try again.');
      }
      setAvatarURL(updatedUser.avatar_url ?? secureURL);
      await setAuth(token, updatedUser);
    } catch (err) {
      Alert.alert('Upload failed', userFacingApiMessage(err));
    } finally {
      setAvatarUploading(false);
    }
  };

  const handleEditAvatar = () => {
    Alert.alert('Update avatar', 'Choose a source', [
      {text: 'Cancel', style: 'cancel'},
      {
        text: 'Camera',
        onPress: () => {
          void handlePickAndUploadAvatar('camera');
        },
      },
      {
        text: 'Photo Library',
        onPress: () => {
          void handlePickAndUploadAvatar('library');
        },
      },
    ]);
  };

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
    } catch (err) {
      if (__DEV__) {
        console.warn('[Account] getPlayerProfile failed', err);
      }
      if (
        err instanceof ApiRequestError &&
        (err.status === 401 || err.status === 403)
      ) {
        await authLogout();
        dispatchResetToMainProfileTab();
        return;
      }
      Alert.alert('Something went wrong', userFacingApiMessage(err));
    } finally {
      setPlayerModeLoading(false);
    }
  };

  const mainContent = (
    <>
      <View style={styles.headerRightRow}>
        <View />
        <TouchableOpacity
          style={styles.headerGear}
          onPress={() => rootNavigate('Settings')}
          accessibilityRole="button"
          accessibilityLabel="Open settings">
          <Ionicons name="settings" size={18} color="#c7f349" />
        </TouchableOpacity>
      </View>

      {!isLoggedIn ? (
        <View style={styles.guestAuthFill}>
          <ProfileGuestAuth returnToDebate={profileAuthReturnToDebate} />
        </View>
      ) : (
        <ScrollView
          style={styles.tabContent}
          showsVerticalScrollIndicator={false}>
          <View style={styles.profileSummary}>
            <View style={styles.profileAvatarWrap}>
              {avatarURL ? (
                <Image source={{uri: avatarURL}} style={styles.avatar} />
              ) : (
                <View style={styles.avatarPlaceholder}>
                  <Ionicons name="person" size={30} color="#94a3b8" />
                </View>
              )}
              <TouchableOpacity
                style={styles.avatarEditBtn}
                onPress={handleEditAvatar}
                disabled={avatarUploading}
                accessibilityLabel="Change profile photo">
                {avatarUploading ? (
                  <ActivityIndicator size="small" color="#c7f349" />
                ) : (
                  <Ionicons name="camera" size={18} color="#c7f349" />
                )}
              </TouchableOpacity>
            </View>
          </View>

          <View style={styles.section}>
            <Text style={styles.accountHeading}>Account</Text>

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
        </ScrollView>
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
  headerRightRow: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingTop: 10,
    paddingBottom: 4,
  },
  headerGear: {
    width: 28,
    height: 28,
    borderRadius: 14,
    alignItems: 'center',
    justifyContent: 'center',
  },
  profileSummary: {
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
    overflow: 'visible',
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
    position: 'absolute',
    right: -8,
    bottom: -8,
    width: 32,
    height: 32,
    borderRadius: 8,
    backgroundColor: '#111827',
    borderWidth: 1,
    borderColor: '#1f2937',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1,
  },
  guestAuthFill: {
    flex: 1,
    minHeight: 400,
  },
  tabContent: {
    flex: 1,
    padding: 16,
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
});
