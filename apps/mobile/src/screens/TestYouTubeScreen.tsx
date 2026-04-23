import React, {useCallback, useMemo, useState} from 'react';
import {ActivityIndicator, StyleSheet, Text, View} from 'react-native';
import {useQuery} from '@tanstack/react-query';
import PagerView from 'react-native-pager-view';
import YoutubeStorySlide, {
  type YoutubeStoryResult,
} from '../components/YoutubeStorySlide';
import {
  FC_BARCELONA_CHANNEL_ID,
  formatLocalYmd,
  isSearchVideoItem,
  searchVideos,
} from '../services/youtubeDataApi';

/** Aligned with App.tsx `SHELL_MATCHES_BG` (matches / test tab shell). */
const SHELL_TEST_BG = '#0B0E14';

const MAX_RESULTS = 10;

/**
 * FC Barcelona official channel only (@FCBarcelona Shorts), local **today**, oldest first.
 * https://www.youtube.com/@FCBarcelona/shorts
 */
export default function TestYouTubeScreen() {
  const [page, setPage] = useState(0);

  const todayKey = formatLocalYmd(new Date());

  const {data, isPending, isError, error, refetch, isRefetching} = useQuery({
    queryKey: [
      'youtubeDataSearch',
      FC_BARCELONA_CHANNEL_ID,
      MAX_RESULTS,
      'shorts',
      'today',
      todayKey,
    ],
    queryFn: () =>
      searchVideos({
        q: '',
        channelId: FC_BARCELONA_CHANNEL_ID,
        order: 'date',
        maxResults: MAX_RESULTS,
        shortsOnly: true,
        publishedOnLocalDay: new Date(),
      }),
  });

  const slides: YoutubeStoryResult[] = useMemo(() => {
    const items = data?.items?.filter(isSearchVideoItem) ?? [];
    return items.map(item => ({
      id: {kind: item.id.kind, videoId: item.id.videoId},
      snippet: {
        channelTitle: item.snippet.channelTitle,
        title: item.snippet.title,
      },
    }));
  }, [data?.items]);

  const onPageSelected = useCallback((e: {nativeEvent: {position: number}}) => {
    setPage(e.nativeEvent.position);
  }, []);

  if (isPending) {
    return (
      <View style={[styles.root, styles.centered]}>
        <ActivityIndicator size="large" color="#C6FF00" />
        <Text style={styles.hint}>Loading YouTube search…</Text>
      </View>
    );
  }

  if (isError) {
    return (
      <View style={[styles.root, styles.centered]}>
        <Text style={styles.errorTitle}>Search failed</Text>
        <Text style={styles.errorBody}>
          {error instanceof Error ? error.message : 'Unknown error'}
        </Text>
        <Text style={styles.retry} onPress={() => void refetch()}>
          {isRefetching ? 'Retrying…' : 'Retry'}
        </Text>
      </View>
    );
  }

  if (slides.length === 0) {
    return (
      <View style={[styles.root, styles.centered]}>
        <Text style={styles.hint}>No embeddable videos returned.</Text>
        <Text style={styles.retry} onPress={() => void refetch()}>
          Refetch
        </Text>
      </View>
    );
  }

  return (
    <View style={styles.root} testID="TestYouTubeScreen">
      <View style={styles.pagerChrome} pointerEvents="none">
        <Text style={styles.pageLabel}>
          {page + 1} / {slides.length}
        </Text>
        <Text style={styles.hint} numberOfLines={3}>
          {`@FCBarcelona (channel) · #shorts · today ${todayKey} · oldest→newest`}
        </Text>
      </View>
      <PagerView
        style={styles.pager}
        initialPage={0}
        onPageSelected={onPageSelected}
        overdrag
        key={slides.map(s => s.id.videoId).join(',')}>
        {slides.map(result => (
          <View key={result.id.videoId} style={styles.page} collapsable={false}>
            <YoutubeStorySlide result={result} />
          </View>
        ))}
      </PagerView>
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: SHELL_TEST_BG,
  },
  centered: {
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  pager: {
    flex: 1,
  },
  page: {
    flex: 1,
  },
  pagerChrome: {
    position: 'absolute',
    top: 8,
    left: 16,
    right: 16,
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
    color: '#C6FF00',
    fontSize: 15,
    fontWeight: '600',
  },
});
