import React, {useCallback, useEffect, useRef, useState} from 'react';
import {
  ActivityIndicator,
  Image,
  Pressable,
  StyleSheet,
  View,
} from 'react-native';
import {Video, ResizeMode, AVPlaybackStatus} from 'expo-av';
import {Ionicons} from '@expo/vector-icons';
import type {FanStory} from '../services/matchShortsApi';
import {FAN_STORY_PHOTO_DURATION_MS} from '../services/matchShortsApi';

type Props = {
  story: FanStory;
  isActive: boolean;
  onFinished: () => void;
  onPlaybackStart?: () => void;
  onReport?: (story: FanStory) => void;
};

export default function FanStorySlide({
  story,
  isActive,
  onFinished,
  onPlaybackStart,
  onReport,
}: Props) {
  const finishedRef = useRef(onFinished);
  const onPlaybackStartRef = useRef(onPlaybackStart);
  finishedRef.current = onFinished;
  onPlaybackStartRef.current = onPlaybackStart;
  const calledRef = useRef(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    calledRef.current = false;
    setLoading(true);
  }, [story.id]);

  const handleFinished = useCallback(() => {
    if (calledRef.current) return;
    calledRef.current = true;
    finishedRef.current();
  }, []);

  useEffect(() => {
    if (!isActive || story.content_type !== 'photo') {
      return;
    }
    onPlaybackStartRef.current?.();
    const timer = setTimeout(handleFinished, FAN_STORY_PHOTO_DURATION_MS);
    return () => clearTimeout(timer);
  }, [handleFinished, isActive, story.content_type, story.id]);

  const onVideoStatus = useCallback(
    (status: AVPlaybackStatus) => {
      if (!status.isLoaded) {
        return;
      }
      if (loading && status.isLoaded) {
        setLoading(false);
      }
      if (status.isPlaying) {
        onPlaybackStartRef.current?.();
      }
      if (status.didJustFinish) {
        handleFinished();
      }
    },
    [handleFinished, loading],
  );

  return (
    <View style={styles.root}>
      {story.content_type === 'video' ? (
        <Video
          source={{uri: story.media_url}}
          style={styles.media}
          resizeMode={ResizeMode.COVER}
          shouldPlay={isActive}
          isLooping={false}
          onPlaybackStatusUpdate={onVideoStatus}
        />
      ) : (
        <Image
          source={{uri: story.media_url}}
          style={styles.media}
          resizeMode="cover"
          onLoad={() => {
            setLoading(false);
            onPlaybackStartRef.current?.();
          }}
        />
      )}
      {loading ? (
        <View style={styles.loader}>
          <ActivityIndicator color="#fff" size="large" />
        </View>
      ) : null}
      {onReport ? (
        <Pressable
          style={styles.reportBtn}
          onPress={() => onReport(story)}
          hitSlop={12}
          accessibilityRole="button"
          accessibilityLabel="Report story">
          <Ionicons name="flag-outline" size={22} color="#fff" />
        </Pressable>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: '#000',
  },
  media: {
    ...StyleSheet.absoluteFillObject,
  },
  loader: {
    ...StyleSheet.absoluteFillObject,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: 'rgba(0,0,0,0.35)',
  },
  reportBtn: {
    position: 'absolute',
    right: 16,
    bottom: 120,
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(0,0,0,0.45)',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 6,
  },
});
