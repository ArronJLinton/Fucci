import React, {useCallback, useState} from 'react';
import {
  ActivityIndicator,
  Alert,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import {useNavigation, useRoute, type RouteProp} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import * as ImagePicker from 'expo-image-picker';
import {Ionicons} from '@expo/vector-icons';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import {useQueryClient} from '@tanstack/react-query';
import type {RootStackParamList} from '../types/navigation';
import {useAuth} from '../context/AuthContext';
import {uploadToCloudinary} from '../services/cloudinaryUpload';
import {createMatchStory} from '../services/matchStoryApi';
import {
  FAN_STORY_VIDEO_MAX_DURATION_MS,
  matchShortsQueryKey,
} from '../services/matchShortsApi';

type Nav = NativeStackNavigationProp<RootStackParamList, 'MatchStoryCapture'>;
type R = RouteProp<RootStackParamList, 'MatchStoryCapture'>;

const VIDEO_MAX_DURATION_SEC = FAN_STORY_VIDEO_MAX_DURATION_MS / 1000;

export default function MatchStoryCaptureScreen() {
  const navigation = useNavigation<Nav>();
  const {params} = useRoute<R>();
  const insets = useSafeAreaInsets();
  const {token, isLoggedIn} = useAuth();
  const queryClient = useQueryClient();
  const [uploading, setUploading] = useState(false);

  const ensureAuth = useCallback(() => {
    if (!isLoggedIn || !token) {
      Alert.alert('Sign in required', 'Please sign in to add a match story.');
      navigation.navigate('Main', {screen: 'Profile'});
      return false;
    }
    return true;
  }, [isLoggedIn, navigation, token]);

  const publishMedia = useCallback(
    async (uri: string, contentType: 'photo' | 'video', fileName?: string) => {
      if (!token || !ensureAuth()) {
        return;
      }
      setUploading(true);
      try {
        const cloudContext =
          contentType === 'video' ? 'match_story_video' : 'match_story_photo';
        const mediaUrl = await uploadToCloudinary(token, cloudContext, {
          uri,
          fileName,
        });
        await createMatchStory(token, {
          scope_id: String(params.matchId),
          team_lookup_key: params.teamLookupKey,
          content_type: contentType,
          media_url: mediaUrl,
        });
        await queryClient.invalidateQueries({
          queryKey: matchShortsQueryKey(params.matchId),
        });
        Alert.alert('Story posted', 'Your story is live for this match.');
        navigation.goBack();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : 'Could not upload your story.';
        Alert.alert('Upload failed', message);
      } finally {
        setUploading(false);
      }
    },
    [ensureAuth, navigation, params.matchId, params.teamLookupKey, queryClient, token],
  );

  const pickFromLibrary = useCallback(async () => {
    if (!ensureAuth()) return;
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted) {
      Alert.alert('Permission needed', 'Allow photo library access to continue.');
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images', 'videos'],
      quality: 0.85,
      videoMaxDuration: VIDEO_MAX_DURATION_SEC,
    });
    if (result.canceled || !result.assets[0]) {
      return;
    }
    const asset = result.assets[0];
    const contentType = asset.type === 'video' ? 'video' : 'photo';
    if (
      contentType === 'video' &&
      asset.duration != null &&
      asset.duration > VIDEO_MAX_DURATION_SEC + 0.5
    ) {
      Alert.alert('Video too long', 'Please choose a video up to 60 seconds.');
      return;
    }
    await publishMedia(asset.uri, contentType, asset.fileName ?? undefined);
  }, [ensureAuth, publishMedia]);

  const captureWithCamera = useCallback(async () => {
    if (!ensureAuth()) return;
    const perm = await ImagePicker.requestCameraPermissionsAsync();
    if (!perm.granted) {
      Alert.alert('Permission needed', 'Allow camera access to continue.');
      return;
    }
    const result = await ImagePicker.launchCameraAsync({
      mediaTypes: ['images', 'videos'],
      quality: 0.85,
      videoMaxDuration: VIDEO_MAX_DURATION_SEC,
    });
    if (result.canceled || !result.assets[0]) {
      return;
    }
    const asset = result.assets[0];
    const contentType = asset.type === 'video' ? 'video' : 'photo';
    await publishMedia(asset.uri, contentType, asset.fileName ?? undefined);
  }, [ensureAuth, publishMedia]);

  return (
    <View style={[styles.root, {paddingTop: insets.top + 12}]}>
      <View style={styles.header}>
        <Pressable onPress={() => navigation.goBack()} hitSlop={12}>
          <Ionicons name="close" size={28} color="#111" />
        </Pressable>
        <Text style={styles.title}>Add story</Text>
        <View style={{width: 28}} />
      </View>
      <Text style={styles.subtitle}>{params.teamDisplayName}</Text>
      <Text style={styles.hint}>
        Photos and videos up to 60 seconds are shared in this match story.
      </Text>

      <Pressable
        style={[styles.actionBtn, uploading && styles.actionBtnDisabled]}
        disabled={uploading}
        onPress={captureWithCamera}>
        <Ionicons name="camera-outline" size={22} color="#111" />
        <Text style={styles.actionText}>Camera</Text>
      </Pressable>

      <Pressable
        style={[styles.actionBtn, uploading && styles.actionBtnDisabled]}
        disabled={uploading}
        onPress={pickFromLibrary}>
        <Ionicons name="images-outline" size={22} color="#111" />
        <Text style={styles.actionText}>Photo library</Text>
      </Pressable>

      {uploading ? (
        <View style={styles.uploadingRow}>
          <ActivityIndicator color="#111" />
          <Text style={styles.uploadingText}>Uploading…</Text>
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: '#fff',
    paddingHorizontal: 20,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  title: {
    fontSize: 18,
    fontWeight: '700',
    color: '#111',
  },
  subtitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#222',
    marginBottom: 8,
  },
  hint: {
    fontSize: 14,
    color: '#666',
    marginBottom: 24,
    lineHeight: 20,
  },
  actionBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    borderWidth: 1,
    borderColor: '#ddd',
    borderRadius: 14,
    paddingVertical: 16,
    paddingHorizontal: 16,
    marginBottom: 12,
  },
  actionBtnDisabled: {
    opacity: 0.6,
  },
  actionText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#111',
  },
  uploadingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    marginTop: 8,
  },
  uploadingText: {
    fontSize: 14,
    color: '#444',
  },
});
