import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import {useQuery} from '@tanstack/react-query';
import PagerView from 'react-native-pager-view';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import {
  useNavigation,
  useRoute,
  type RouteProp,
} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {Ionicons} from '@expo/vector-icons';
import SnapchatStorySlide from '../components/SnapchatStorySlide';
import {
  fetchSnapchatUserStories,
  type SnapchatStoryItem,
} from '../services/snapchatStoriesApi';
import type {RootStackParamList} from '../types/navigation';
import {userFacingApiMessage} from '../services/api';

const BG = '#0B0E14';
const LIME = '#C6FF00';
/** Photo snaps auto-advance after this many seconds. */
const PHOTO_STORY_SEC = 5;
const SNAP_MEDIA_IMAGE = 0;

type Nav = NativeStackNavigationProp<
  RootStackParamList,
  'MatchSnapchatStories'
>;
type R = RouteProp<RootStackParamList, 'MatchSnapchatStories'>;

export default function MatchSnapchatStoriesScreen() {
  const navigation = useNavigation<Nav>();
  const {params} = useRoute<R>();
  const {snapchatUsername, teamDisplayName} = params;
  const insets = useSafeAreaInsets();
  const [page, setPage] = useState(0);
  const pagerRef = useRef<PagerView>(null);
  const pageRef = useRef(0);

  useEffect(() => {
    pageRef.current = page;
  }, [page]);

  const {data, isPending, isError, error, refetch, isRefetching} = useQuery({
    queryKey: ['snapchatUserStories', snapchatUsername],
    queryFn: () => fetchSnapchatUserStories(snapchatUsername),
  });

  const slides: SnapchatStoryItem[] = useMemo(() => {
    const raw = data?.stories ?? [];
    return [...raw]
      .filter(s => Boolean(s?.snapUrls?.mediaUrl))
      .sort((a, b) => a.snapIndex - b.snapIndex);
  }, [data?.stories]);

  const profileTitle = data?.user?.name ?? teamDisplayName;
  const profileUser = data?.user?.username ?? snapchatUsername;

  const onPageSelected = useCallback((e: {nativeEvent: {position: number}}) => {
    setPage(e.nativeEvent.position);
  }, []);

  const goToNext = useCallback(() => {
    const p = pageRef.current;
    if (p >= slides.length - 1) {
      return;
    }
    pagerRef.current?.setPage(p + 1);
  }, [slides.length]);

  const goToPrev = useCallback(() => {
    const p = pageRef.current;
    if (p <= 0) {
      return;
    }
    pagerRef.current?.setPage(p - 1);
  }, []);

  /** Image snaps: advance after PHOTO_STORY_SEC (restarts on manual swipe). */
  useEffect(() => {
    if (slides.length === 0) {
      return;
    }
    const p = page;
    const story = slides[p];
    if (!story) {
      return;
    }
    if (story.snapMediaType !== SNAP_MEDIA_IMAGE) {
      return;
    }
    const t = setTimeout(() => {
      if (pageRef.current !== p) {
        return;
      }
      if (p < slides.length - 1) {
        pagerRef.current?.setPage(p + 1);
      } else {
        navigation.goBack();
      }
    }, PHOTO_STORY_SEC * 1000);
    return () => clearTimeout(t);
  }, [page, slides, navigation]);

  const onVideoFinished = useCallback(() => {
    const p = pageRef.current;
    if (p >= slides.length - 1) {
      navigation.goBack();
      return;
    }
    goToNext();
  }, [goToNext, navigation, slides.length]);

  if (isPending) {
    return (
      <View style={[styles.root, styles.centered]}>
        <ActivityIndicator size="large" color={LIME} />
        <Text style={styles.hint}>Loading Snapchat…</Text>
      </View>
    );
  }

  if (isError) {
    return (
      <View style={[styles.root, styles.centered]}>
        <Text style={styles.errorTitle}>Couldn’t load stories</Text>
        <Text style={styles.errorBody}>{userFacingApiMessage(error)}</Text>
        <Text style={styles.retry} onPress={() => void refetch()}>
          {isRefetching ? 'Retrying…' : 'Retry'}
        </Text>
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

  if (slides.length === 0) {
    return (
      <View style={[styles.root, styles.centered]}>
        <Text style={styles.hint}>
          No story media for this account right now.
        </Text>
        <Text style={styles.retry} onPress={() => void refetch()}>
          Refresh
        </Text>
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
    <View style={styles.root} testID="MatchSnapchatStoriesScreen">
      <Pressable
        style={[styles.closeFab, {top: insets.top + 8}]}
        onPress={() => navigation.goBack()}
        hitSlop={12}
        accessibilityRole="button"
        accessibilityLabel="Close">
        <Ionicons name="close" size={26} color="#111" />
      </Pressable>

      <View
        style={[styles.pagerChrome, {paddingTop: insets.top + 6}]}
        pointerEvents="none">
        <Text style={styles.pageLabel}>
          {page + 1} / {slides.length}
        </Text>
        <Text style={styles.hint} numberOfLines={2}>
          @{profileUser} · {teamDisplayName}
        </Text>
      </View>

      <View style={styles.pagerWrap}>
        <PagerView
          ref={pagerRef}
          style={styles.pager}
          initialPage={0}
          orientation="vertical"
          onPageSelected={onPageSelected}
          overdrag
          key={slides
            .map(s => String(s.snapIndex) + s.snapUrls?.mediaUrl)
            .join(',')}>
          {slides.map((story, index) => (
            <View
              key={`${story.snapIndex}-${story.snapUrls?.mediaUrl ?? ''}`}
              style={styles.page}
              collapsable={false}>
              <SnapchatStorySlide
                story={story}
                profileTitle={profileTitle}
                profileUsername={profileUser}
                isActive={page === index}
                onVideoFinished={onVideoFinished}
              />
            </View>
          ))}
        </PagerView>

        <View
          style={[
            styles.tapZones,
            {top: insets.top + 52, bottom: 100},
          ]}
          pointerEvents="box-none">
          <Pressable
            style={styles.tapHalf}
            onPress={goToPrev}
            accessibilityRole="button"
            accessibilityLabel="Previous story"
            hitSlop={0}
          />
          <Pressable
            style={styles.tapHalf}
            onPress={goToNext}
            accessibilityRole="button"
            accessibilityLabel="Next story"
            hitSlop={0}
          />
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: BG,
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
    flexDirection: 'row',
  },
  tapHalf: {
    flex: 1,
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
  pager: {
    flex: 1,
  },
  page: {
    flex: 1,
  },
  pagerChrome: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    zIndex: 2,
    alignItems: 'center',
  },
  pageLabel: {
    color: 'rgba(255,255,255,0.85)',
    fontSize: 12,
    fontWeight: '700',
    letterSpacing: 0.5,
  },
  hint: {
    marginTop: 4,
    color: 'rgba(255,255,255,0.45)',
    fontSize: 10,
    textAlign: 'center',
  },
  errorTitle: {
    color: 'rgba(255,255,255,0.9)',
    fontSize: 16,
    fontWeight: '700',
    marginBottom: 8,
    textAlign: 'center',
  },
  errorBody: {
    color: 'rgba(255,255,255,0.6)',
    fontSize: 13,
    textAlign: 'center',
  },
  retry: {
    marginTop: 16,
    color: LIME,
    fontSize: 15,
    fontWeight: '600',
  },
});
