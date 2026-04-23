import React, { useCallback, useEffect, useRef } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Image,
  Linking,
  Pressable,
} from 'react-native';
import { Video, ResizeMode, type AVPlaybackStatus } from 'expo-av';
import type { SnapchatStoryItem } from '../services/snapchatStoriesApi';

type Props = {
  story: SnapchatStoryItem;
  profileTitle: string;
  profileUsername: string;
  /** When false, video is paused (off-screen in pager). */
  isActive: boolean;
  /** Fired once when a video snap finishes (non-looping). */
  onVideoFinished?: () => void;
};

/** RapidAPI sample: 0 = image story, 1 = video. */
const SNAP_IMAGE = 0;

function titleFor(story: SnapchatStoryItem, profileTitle: string): string {
  const t = story.snapTitle;
  if (typeof t === 'string' && t.trim().length > 0) {
    return t;
  }
  return profileTitle;
}

export default function SnapchatStorySlide({
  story,
  profileTitle,
  profileUsername,
  isActive,
  onVideoFinished,
}: Props) {
  const mediaUrl = story.snapUrls?.mediaUrl;
  const isVideo = story.snapMediaType !== SNAP_IMAGE;
  const videoRef = useRef<Video | null>(null);
  const hasEndedRef = useRef(false);

  useEffect(() => {
    hasEndedRef.current = false;
  }, [story.snapIndex, mediaUrl]);

  useEffect(() => {
    return () => {
      void videoRef.current?.unloadAsync();
    };
  }, []);

  const onPlaybackStatusUpdate = useCallback(
    (s: AVPlaybackStatus) => {
      if (!s.isLoaded || !onVideoFinished) {
        return;
      }
      if (hasEndedRef.current) {
        return;
      }
      if (s.didJustFinish) {
        hasEndedRef.current = true;
        onVideoFinished();
      }
    },
    [onVideoFinished],
  );

  const openProfile = () => {
    const u = `https://www.snapchat.com/@${encodeURIComponent(profileUsername)}`;
    void Linking.openURL(u);
  };

  if (!mediaUrl) {
    return (
      <View style={styles.container}>
        <Text style={styles.muted}>No media URL for this snap.</Text>
        <Pressable onPress={openProfile} style={styles.link} hitSlop={8}>
          <Text style={styles.linkText}>View on Snapchat</Text>
        </Pressable>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      {isVideo ? (
        <Video
          ref={videoRef}
          style={styles.media}
          source={{ uri: mediaUrl }}
          resizeMode={ResizeMode.COVER}
          shouldPlay={isActive}
          isLooping={false}
          isMuted
          useNativeControls={false}
          onPlaybackStatusUpdate={onPlaybackStatusUpdate}
        />
      ) : (
        <Image
          source={{ uri: mediaUrl }}
          style={styles.media}
          resizeMode="cover"
        />
      )}

      <View style={styles.overlay} pointerEvents="box-none">
        <Text style={styles.channel}>{profileUsername}</Text>
        <Text style={styles.title} numberOfLines={3}>
          {titleFor(story, profileTitle)}
        </Text>
        <Pressable
          onPress={openProfile}
          style={styles.fallback}
          hitSlop={8}
          accessibilityRole="link"
          accessibilityLabel="Open profile in Snapchat">
          <Text style={styles.fallbackText}>View on Snapchat</Text>
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    width: '100%',
    backgroundColor: '#000',
  },
  media: {
    ...StyleSheet.absoluteFillObject,
  },
  overlay: {
    position: 'absolute',
    bottom: 80,
    left: 16,
    right: 16,
  },
  channel: {
    color: 'rgba(255,255,255,0.55)',
    fontSize: 12,
    fontWeight: '700',
    marginBottom: 4,
  },
  title: {
    color: '#fff',
    fontSize: 15,
    fontWeight: '600',
  },
  fallback: {
    marginTop: 12,
    alignSelf: 'flex-start',
  },
  fallbackText: {
    color: '#C6FF00',
    fontSize: 14,
    fontWeight: '600',
  },
  muted: {
    color: 'rgba(255,255,255,0.5)',
    textAlign: 'center',
    padding: 24,
  },
  link: { alignSelf: 'center', marginTop: 8 },
  linkText: { color: '#C6FF00', fontSize: 15, fontWeight: '600' },
});
