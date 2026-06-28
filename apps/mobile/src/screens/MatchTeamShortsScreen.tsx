import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import {Animated, Pressable, StyleSheet, Text, View} from 'react-native';
import PagerView from 'react-native-pager-view';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import {
  useNavigation,
  useRoute,
  type RouteProp,
} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {Ionicons} from '@expo/vector-icons';
import YouTubeShortSlide from '../components/YouTubeShortSlide';
import {
  parseYouTubeDurationSeconds,
  type YouTubeShort,
} from '../services/matchShortsApi';
import type {RootStackParamList} from '../types/navigation';

const SHORT_RING_AMBER = '#F5A623';
const PROGRESS_TRACK = 'rgba(255,255,255,0.35)';

type Nav = NativeStackNavigationProp<RootStackParamList, 'MatchTeamShorts'>;
type R = RouteProp<RootStackParamList, 'MatchTeamShorts'>;

export default function MatchTeamShortsScreen() {
  const navigation = useNavigation<Nav>();
  const {params} = useRoute<R>();
  const {shorts} = params;
  const insets = useSafeAreaInsets();
  const [page, setPage] = useState(0);
  const [playbackStarted, setPlaybackStarted] = useState(false);
  const progressAnim = useRef(new Animated.Value(0)).current;
  const pagerRef = useRef<PagerView>(null);
  const pageRef = useRef(0);

  const slides: YouTubeShort[] = useMemo(() => shorts ?? [], [shorts]);
  const currentShort = slides[page];
  const currentDurationMs = useMemo(() => {
    const sec = parseYouTubeDurationSeconds(currentShort?.duration ?? '');
    return Math.max(sec * 1000, 3000);
  }, [currentShort?.duration, currentShort?.video_id]);

  useEffect(() => {
    progressAnim.setValue(0);
    setPlaybackStarted(false);
  }, [page, currentShort?.video_id, progressAnim]);

  useEffect(() => {
    if (!playbackStarted || !currentShort) {
      return;
    }
    progressAnim.setValue(0);
    const anim = Animated.timing(progressAnim, {
      toValue: 1,
      duration: currentDurationMs,
      useNativeDriver: false,
    });
    anim.start();
    return () => {
      anim.stop();
    };
  }, [
    playbackStarted,
    page,
    currentShort?.video_id,
    currentDurationMs,
    progressAnim,
  ]);

  const onPlaybackStart = useCallback(() => {
    setPlaybackStarted(true);
  }, []);

  useEffect(() => {
    pageRef.current = page;
  }, [page]);

  const pagerIdentityKey = useMemo(
    () => slides.map(s => s.video_id).join(','),
    [slides],
  );

  useEffect(() => {
    if (slides.length === 0) {
      pageRef.current = 0;
      setPage(0);
      return;
    }
    setPage(prev => {
      const maxIdx = slides.length - 1;
      if (prev <= maxIdx) {
        pageRef.current = prev;
        return prev;
      }
      const clamped = maxIdx;
      pageRef.current = clamped;
      requestAnimationFrame(() => {
        pagerRef.current?.setPage(clamped);
      });
      return clamped;
    });
  }, [slides.length]);

  useEffect(() => {
    if (slides.length === 0) {
      return;
    }
    pageRef.current = 0;
    setPage(0);
    const id = requestAnimationFrame(() => {
      pagerRef.current?.setPage(0);
    });
    return () => cancelAnimationFrame(id);
  }, [pagerIdentityKey, slides.length]);

  const onPageSelected = useCallback((e: {nativeEvent: {position: number}}) => {
    setPage(e.nativeEvent.position);
  }, []);

  const goToNext = useCallback(() => {
    const p = pageRef.current;
    if (p >= slides.length - 1) {
      navigation.goBack();
      return;
    }
    pagerRef.current?.setPage(p + 1);
  }, [navigation, slides.length]);

  const goToPrev = useCallback(() => {
    const p = pageRef.current;
    if (p <= 0) {
      return;
    }
    pagerRef.current?.setPage(p - 1);
  }, []);

  const onShortFinished = useCallback(() => {
    goToNext();
  }, [goToNext]);

  if (slides.length === 0) {
    return (
      <View style={[styles.root, styles.centered]}>
        <Text style={styles.hint}>No Shorts available right now.</Text>
        <Pressable
          style={[styles.closeFab, {top: insets.top + 8}]}
          onPress={() => navigation.goBack()}
          hitSlop={12}
          accessibilityRole="button"
          accessibilityLabel="Close">
          <Ionicons name="close" size={26} color="#111" />
        </Pressable>
      </View>
    );
  }

  return (
    <View style={styles.root} testID="MatchTeamShortsScreen">
      <Pressable
        style={[styles.closeFab, {top: insets.top + 8}]}
        onPress={() => navigation.goBack()}
        hitSlop={12}
        accessibilityRole="button"
        accessibilityLabel="Close">
        <Ionicons name="close" size={26} color="#111" />
      </Pressable>

      <View
        style={[styles.progressBar, {top: insets.top + 10}]}
        pointerEvents="none">
        {slides.map((_, i) => (
          <View key={i} style={styles.progressTrack}>
            {i < page ? (
              <View style={styles.progressFill} />
            ) : i === page ? (
              <Animated.View
                style={[
                  styles.progressFill,
                  {
                    width: progressAnim.interpolate({
                      inputRange: [0, 1],
                      outputRange: ['0%', '100%'],
                    }),
                  },
                ]}
              />
            ) : null}
          </View>
        ))}
      </View>

      <View style={styles.pagerWrap}>
        <PagerView
          ref={pagerRef}
          style={styles.pager}
          initialPage={0}
          orientation="horizontal"
          onPageSelected={onPageSelected}
          overdrag
          key={pagerIdentityKey}>
          {slides.map((short, index) => (
            <View
              key={short.video_id}
              style={styles.page}
              collapsable={false}>
              <YouTubeShortSlide
                short={short}
                isActive={page === index}
                onFinished={onShortFinished}
                onPlaybackStart={
                  page === index ? onPlaybackStart : undefined
                }
              />
            </View>
          ))}
        </PagerView>

        <View
          style={[
            styles.tapZones,
            {top: insets.top + 52, bottom: 100},
          ]}
          pointerEvents={playbackStarted ? 'box-none' : 'none'}>
          <Pressable
            style={styles.tapEdgeLeft}
            onPress={goToPrev}
            accessibilityRole="button"
            accessibilityLabel="Previous short"
          />
          <Pressable
            style={styles.tapEdgeRight}
            onPress={goToNext}
            accessibilityRole="button"
            accessibilityLabel="Next short"
          />
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: '#000',
  },
  centered: {
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  pagerWrap: {
    flex: 1,
  },
  tapZones: {
    position: 'absolute',
    left: 0,
    right: 0,
    zIndex: 5,
  },
  tapEdgeLeft: {
    position: 'absolute',
    left: 0,
    top: 0,
    bottom: 0,
    width: '32%',
  },
  tapEdgeRight: {
    position: 'absolute',
    right: 0,
    top: 0,
    bottom: 0,
    width: '32%',
  },
  closeFab: {
    position: 'absolute',
    left: 12,
    zIndex: 10,
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: 'rgba(255,255,255,0.94)',
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 1},
    shadowOpacity: 0.2,
    shadowRadius: 3,
    elevation: 4,
  },
  progressBar: {
    position: 'absolute',
    left: 12,
    right: 12,
    zIndex: 9,
    flexDirection: 'row',
    gap: 4,
  },
  progressTrack: {
    flex: 1,
    height: 3,
    borderRadius: 2,
    backgroundColor: PROGRESS_TRACK,
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    width: '100%',
    backgroundColor: SHORT_RING_AMBER,
    borderRadius: 2,
  },
  pager: {
    flex: 1,
  },
  page: {
    flex: 1,
  },
  hint: {
    marginTop: 4,
    color: 'rgba(255,255,255,0.45)',
    fontSize: 10,
    textAlign: 'center',
  },
});
