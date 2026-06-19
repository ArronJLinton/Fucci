import React, {useCallback, useEffect, useRef, useState} from 'react';
import {
  ActivityIndicator,
  StyleSheet,
  View,
} from 'react-native';
import {WebView} from 'react-native-webview';
import {
  parseYouTubeDurationSeconds,
  YOUTUBE_SHORT_PLAYER_BASE_URL,
  youtubeShortPlayerHtml,
  type YouTubeShort,
} from '../services/matchShortsApi';
import {MATCH_CENTER_LIME} from '../constants/matchCenterUi';

type Props = {
  short: YouTubeShort;
  isActive: boolean;
  onFinished: () => void;
  onPlaybackStart?: () => void;
};

export default function YouTubeShortSlide({
  short,
  isActive,
  onFinished,
  onPlaybackStart,
}: Props) {
  const finishedRef = useRef(onFinished);
  const onPlaybackStartRef = useRef(onPlaybackStart);
  finishedRef.current = onFinished;
  onPlaybackStartRef.current = onPlaybackStart;

  /** Guards against handleFinished firing more than once for the same video. */
  const calledRef = useRef(false);
  useEffect(() => {
    calledRef.current = false;
  }, [short.video_id]);

  const handleFinished = useCallback(() => {
    if (calledRef.current) return;
    calledRef.current = true;
    finishedRef.current();
  }, []);

  /** Fallback auto-advance if ENDED never fires. */
  useEffect(() => {
    if (!isActive) {
      return;
    }
    const durationSec = parseYouTubeDurationSeconds(short.duration);
    const waitMs = Math.max((durationSec + 1) * 1000, 5000);
    const timer = setTimeout(handleFinished, waitMs);
    return () => clearTimeout(timer);
  }, [handleFinished, isActive, short.duration, short.video_id]);

  const onMessage = useCallback(
    (event: {nativeEvent: {data: string}}) => {
      try {
        const msg = JSON.parse(event.nativeEvent.data) as {type?: string};
        if (msg.type === 'ready') {
          onPlaybackStartRef.current?.();
        }
        if (msg.type === 'ended') {
          handleFinished();
        }
      } catch {
        /* ignore non-JSON postMessage */
      }
    },
    [handleFinished],
  );

  if (!isActive) {
    return <View style={styles.inactive} />;
  }

  return (
    <View style={styles.root}>
      <WebView
        key={short.video_id}
        source={{
          html: youtubeShortPlayerHtml(short.video_id),
          baseUrl: YOUTUBE_SHORT_PLAYER_BASE_URL,
        }}
        style={styles.webview}
        originWhitelist={['https://*']}
        allowsInlineMediaPlayback
        allowsFullscreenVideo={false}
        mediaPlaybackRequiresUserAction={false}
        javaScriptEnabled
        domStorageEnabled
        scrollEnabled={false}
        bounces={false}
        onMessage={onMessage}
        startInLoadingState
        renderLoading={() => (
          <View style={styles.loading}>
            <ActivityIndicator size="large" color={MATCH_CENTER_LIME} />
          </View>
        )}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: '#000',
  },
  inactive: {
    flex: 1,
    backgroundColor: '#000',
  },
  webview: {
    flex: 1,
    backgroundColor: '#000',
  },
  loading: {
    ...StyleSheet.absoluteFillObject,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#000',
  },
});
