import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import {
  Alert,
  Animated,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import {useQueryClient} from '@tanstack/react-query';
import PagerView from 'react-native-pager-view';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import {
  useFocusEffect,
  useNavigation,
  useRoute,
  type RouteProp,
} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {Ionicons} from '@expo/vector-icons';
import {
  lockAppPortrait,
  unlockAppOrientation,
} from '../utils/screenOrientation';
import YouTubeShortSlide from '../components/YouTubeShortSlide';
import FanStorySlide from '../components/FanStorySlide';
import {useAuth} from '../context/AuthContext';
import {rootNavigateToProfileAuth} from '../navigation/authNavigationActions';
import {
  buildStorySlides,
  matchShortsQueryKey,
  type FanStory,
  type StorySlide,
} from '../services/matchShortsApi';
import {deleteMatchStory, reportMatchStory} from '../services/matchStoryApi';
import type {RootStackParamList} from '../types/navigation';

const SHORT_RING_AMBER = '#F5A623';
const PROGRESS_TRACK = 'rgba(255,255,255,0.35)';
const PROGRESS_BAR_HEIGHT = 3;
const HEADER_TOP_GAP = 8;
const HEADER_SECTION_GAP = 10;
/** Original FAB was 44; 25% smaller → 33. */
const CLOSE_BUTTON_SIZE = 33;
const CLOSE_ICON_SIZE = 20;
const ADD_ICON_SIZE = 22;
/** Progress bars keep a wider inset; FABs sit closer to the corners. */
const HORIZONTAL_CHROME_INSET = 12;
const CORNER_CHROME_INSET = 6;

export function shortsHeaderLayout(insetsTop: number) {
  const progressBarTop = insetsTop + HEADER_TOP_GAP;
  const metaRowTop =
    progressBarTop + PROGRESS_BAR_HEIGHT + HEADER_SECTION_GAP;
  const chromeBottom = metaRowTop + CLOSE_BUTTON_SIZE;
  return {
    progressBarTop,
    closeButtonTop: metaRowTop,
    metaRowTop,
    chromeBottom,
  };
}

type Nav = NativeStackNavigationProp<RootStackParamList, 'MatchTeamShorts'>;
type R = RouteProp<RootStackParamList, 'MatchTeamShorts'>;

export default function MatchTeamShortsScreen() {
  const navigation = useNavigation<Nav>();
  const {params} = useRoute<R>();
  const queryClient = useQueryClient();
  const {token, isLoggedIn, user} = useAuth();
  const insets = useSafeAreaInsets();
  const headerLayout = shortsHeaderLayout(insets.top);
  const [page, setPage] = useState(0);
  const [playbackStarted, setPlaybackStarted] = useState(false);
  const [removedFanIds, setRemovedFanIds] = useState<Set<string>>(new Set());
  const progressAnim = useRef(new Animated.Value(0)).current;
  const pagerRef = useRef<PagerView>(null);
  const pageRef = useRef(0);

  const userStories = useMemo(
    () =>
      (params.userStories ?? []).filter(story => !removedFanIds.has(story.id)),
    [params.userStories, removedFanIds],
  );
  const youtubeShorts = params.youtubeShorts ?? params.shorts ?? [];

  const slides: StorySlide[] = useMemo(
    () => buildStorySlides(userStories, youtubeShorts),
    [userStories, youtubeShorts],
  );

  const currentSlide = slides[page];
  const currentDurationMs = currentSlide?.durationMs ?? 3000;

  useFocusEffect(
    useCallback(() => {
      unlockAppOrientation();
      return lockAppPortrait;
    }, []),
  );

  useEffect(() => {
    progressAnim.setValue(0);
    setPlaybackStarted(false);
  }, [page, currentSlide?.slideKey, progressAnim]);

  useEffect(() => {
    if (!playbackStarted || !currentSlide) {
      return;
    }
    if (currentSlide.kind === 'fan' && currentSlide.story.content_type === 'video') {
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
    currentSlide,
    currentDurationMs,
    progressAnim,
  ]);

  const onPlaybackStart = useCallback(() => {
    setPlaybackStarted(true);
  }, []);

  const pagerIdentityKey = useMemo(
    () => slides.map(s => s.slideKey).join('|'),
    [slides],
  );

  // PagerView remounts via key={pagerIdentityKey} at initialPage 0. Keep React
  // page state in sync or isActive / goToNext break after report/delete.
  useEffect(() => {
    if (slides.length === 0) {
      pageRef.current = 0;
      setPage(0);
      return;
    }
    pageRef.current = 0;
    setPage(0);
  }, [pagerIdentityKey, slides.length]);

  const goToNext = useCallback(() => {
    const next = pageRef.current + 1;
    if (next >= slides.length) {
      navigation.goBack();
      return;
    }
    pagerRef.current?.setPage(next);
  }, [navigation, slides.length]);

  const goToPrev = useCallback(() => {
    const prev = pageRef.current - 1;
    if (prev < 0) {
      return;
    }
    pagerRef.current?.setPage(prev);
  }, []);

  const onSlideFinished = useCallback(() => {
    goToNext();
  }, [goToNext]);

  const onPageSelected = useCallback(
    (e: {nativeEvent: {position: number}}) => {
      pageRef.current = e.nativeEvent.position;
      setPage(e.nativeEvent.position);
    },
    [],
  );

  const openCapture = useCallback(() => {
    if (!isLoggedIn) {
      Alert.alert('Sign in required', 'Please sign in to add a match story.');
      // Login stack was removed; auth lives on the Profile tab.
      rootNavigateToProfileAuth();
      return;
    }
    if (params.matchId == null || !params.teamLookupKey) {
      return;
    }
    navigation.navigate('MatchStoryCapture', {
      matchId: params.matchId,
      teamLookupKey: params.teamLookupKey,
      teamDisplayName: params.teamDisplayName,
    });
  }, [isLoggedIn, navigation, params.matchId, params.teamDisplayName, params.teamLookupKey]);

  const onReportStory = useCallback(
    (story: FanStory) => {
      Alert.alert('Report story?', 'This story will be reviewed by our team.', [
        {text: 'Cancel', style: 'cancel'},
        {
          text: 'Report',
          style: 'destructive',
          onPress: async () => {
            if (!token) {
              Alert.alert('Sign in required', 'Please sign in to report a story.');
              return;
            }
            try {
              await reportMatchStory(token, story.id);
              setRemovedFanIds(prev => new Set(prev).add(story.id));
              if (params.matchId != null) {
                await queryClient.invalidateQueries({
                  queryKey: matchShortsQueryKey(params.matchId),
                });
              }
            } catch {
              Alert.alert('Could not report', 'Please try again.');
            }
          },
        },
      ]);
    },
    [params.matchId, queryClient, token],
  );

  const onDeleteStory = useCallback(
    (story: FanStory) => {
      Alert.alert(
        'Delete story?',
        'This story will be removed from the match feed.',
        [
          {text: 'Cancel', style: 'cancel'},
          {
            text: 'Delete',
            style: 'destructive',
            onPress: async () => {
              if (!token) {
                Alert.alert('Sign in required', 'Please sign in to delete your story.');
                return;
              }
              try {
                await deleteMatchStory(token, story.id);
                setRemovedFanIds(prev => new Set(prev).add(story.id));
                if (params.matchId != null) {
                  await queryClient.invalidateQueries({
                    queryKey: matchShortsQueryKey(params.matchId),
                  });
                }
              } catch {
                Alert.alert('Could not delete', 'Please try again.');
              }
            },
          },
        ],
      );
    },
    [params.matchId, queryClient, token],
  );

  const canAddStory =
    params.matchId != null && Boolean(params.teamLookupKey);

  if (slides.length === 0) {
    return (
      <View style={[styles.root, styles.centered]}>
        <Text style={styles.emptyTitle}>No stories yet</Text>
        <Text style={styles.emptyHint}>
          Be the first to share a photo or video for {params.teamDisplayName}.
        </Text>
        {canAddStory ? (
          <Pressable style={styles.addStoryBtn} onPress={openCapture}>
            <Ionicons name="add" size={ADD_ICON_SIZE} color="#111" />
            <Text style={styles.addStoryBtnText}>Add story</Text>
          </Pressable>
        ) : null}
        <Pressable
          style={[
            styles.closeFab,
            styles.closeFabAbsolute,
            {
              top: headerLayout.closeButtonTop,
              left: CORNER_CHROME_INSET + insets.left,
            },
          ]}
          onPress={() => navigation.goBack()}
          hitSlop={12}
          accessibilityRole="button"
          accessibilityLabel="Close">
          <Ionicons name="close" size={CLOSE_ICON_SIZE} color="#111" />
        </Pressable>
      </View>
    );
  }

  return (
    <View style={styles.root} testID="MatchTeamShortsScreen">
      <View style={styles.pagerWrap}>
        <PagerView
          ref={pagerRef}
          style={styles.pager}
          initialPage={0}
          orientation="horizontal"
          onPageSelected={onPageSelected}
          overdrag
          key={pagerIdentityKey}>
          {slides.map((slide, index) => (
            <View key={slide.slideKey} style={styles.page} collapsable={false}>
              {slide.kind === 'fan' ? (
                <FanStorySlide
                  story={slide.story}
                  isActive={page === index}
                  onFinished={onSlideFinished}
                  onPlaybackStart={
                    page === index ? onPlaybackStart : undefined
                  }
                  onReport={
                    user?.id === slide.story.user_id ? undefined : onReportStory
                  }
                  onDelete={
                    user?.id === slide.story.user_id ? onDeleteStory : undefined
                  }
                />
              ) : (
                <YouTubeShortSlide
                  short={slide.short}
                  isActive={page === index}
                  onFinished={onSlideFinished}
                  onPlaybackStart={
                    page === index ? onPlaybackStart : undefined
                  }
                />
              )}
            </View>
          ))}
        </PagerView>

        <View
          style={[
            styles.tapZones,
            {top: headerLayout.chromeBottom + 8, bottom: 100},
          ]}
          pointerEvents={playbackStarted ? 'box-none' : 'none'}>
          <Pressable
            style={styles.tapEdgeLeft}
            onPress={goToPrev}
            accessibilityRole="button"
            accessibilityLabel="Previous story"
          />
          <Pressable
            style={styles.tapEdgeRight}
            onPress={goToNext}
            accessibilityRole="button"
            accessibilityLabel="Next story"
          />
        </View>
      </View>

      <View
        style={[
          styles.progressBar,
          {
            top: headerLayout.progressBarTop,
            left: HORIZONTAL_CHROME_INSET + insets.left,
            right: HORIZONTAL_CHROME_INSET + insets.right,
          },
        ]}
        pointerEvents="none">
        {slides.map((slide, i) => (
          <View key={slide.slideKey} style={styles.progressTrack}>
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

      <Pressable
        style={[
          styles.closeFab,
          styles.closeFabAbsolute,
          {
            top: headerLayout.metaRowTop,
            left: CORNER_CHROME_INSET + insets.left,
          },
        ]}
        onPress={() => navigation.goBack()}
        hitSlop={12}
        accessibilityRole="button"
        accessibilityLabel="Close">
        <Ionicons name="close" size={CLOSE_ICON_SIZE} color="#111" />
      </Pressable>

      {canAddStory ? (
        <Pressable
          style={[
            styles.addFab,
            styles.addFabAbsolute,
            {
              top: headerLayout.metaRowTop,
              right: CORNER_CHROME_INSET + insets.right,
            },
          ]}
          onPress={openCapture}
          hitSlop={12}
          accessibilityRole="button"
          accessibilityLabel="Add story">
          <Ionicons name="add" size={ADD_ICON_SIZE} color="#111" />
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
  centered: {
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  pagerWrap: {
    ...StyleSheet.absoluteFillObject,
  },
  pager: {
    flex: 1,
  },
  page: {
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
  progressBar: {
    position: 'absolute',
    zIndex: 9,
    flexDirection: 'row',
    gap: 4,
  },
  progressTrack: {
    flex: 1,
    height: PROGRESS_BAR_HEIGHT,
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
  closeFab: {
    width: CLOSE_BUTTON_SIZE,
    height: CLOSE_BUTTON_SIZE,
    borderRadius: CLOSE_BUTTON_SIZE / 2,
    backgroundColor: 'rgba(255,255,255,0.94)',
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 1},
    shadowOpacity: 0.2,
    shadowRadius: 3,
    elevation: 4,
  },
  closeFabAbsolute: {
    position: 'absolute',
    zIndex: 10,
  },
  addFab: {
    width: CLOSE_BUTTON_SIZE,
    height: CLOSE_BUTTON_SIZE,
    borderRadius: CLOSE_BUTTON_SIZE / 2,
    backgroundColor: SHORT_RING_AMBER,
    alignItems: 'center',
    justifyContent: 'center',
  },
  addFabAbsolute: {
    position: 'absolute',
    zIndex: 10,
  },
  emptyTitle: {
    color: '#fff',
    fontSize: 18,
    fontWeight: '700',
    marginBottom: 8,
    textAlign: 'center',
  },
  emptyHint: {
    color: 'rgba(255,255,255,0.7)',
    fontSize: 14,
    textAlign: 'center',
    marginBottom: 20,
    lineHeight: 20,
  },
  addStoryBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    backgroundColor: SHORT_RING_AMBER,
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderRadius: 22,
    marginBottom: 16,
  },
  addStoryBtnText: {
    color: '#111',
    fontWeight: '700',
    fontSize: 15,
  },
});
