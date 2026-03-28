import React, {useCallback, useMemo, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  Image,
  Linking,
  ActivityIndicator,
} from 'react-native';
import {LinearGradient} from 'expo-linear-gradient';
import {Gesture, GestureDetector} from 'react-native-gesture-handler';
import Animated, {
  runOnJS,
  useAnimatedStyle,
  useSharedValue,
  withSpring,
} from 'react-native-reanimated';
import {Ionicons} from '@expo/vector-icons';
import {useQuery} from '@tanstack/react-query';
import type {DebateSummary, DebateResponse} from '../types/debate';
import type {Match} from '../types/match';
import {fetchDebateById, setCardVote} from '../services/debate';
import {rootNavigate} from '../navigation/rootNavigation';

const BG = '#0B0E14';
const LIME = '#C6FF00';
const CARD = '#1A1F2E';
const TEXT = '#FFFFFF';
const MUTED = '#8B92A5';
const RED_X = '#FF3B30';
const HERO_IMAGE_URI =
  'https://images.unsplash.com/photo-1574629810360-7efbbe195018?w=1200&q=80';

const SWIPE_THRESHOLD = 80;

function debatePillLabel(debateType: string): string {
  return debateType === 'post_match' ? 'CONTROVERSY' : 'PRE-MATCH';
}

function formatVoteCount(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
  return String(n);
}

function relativeTimeLabel(iso: string): string {
  const t = new Date(iso).getTime();
  const diff = Date.now() - t;
  const h = Math.floor(diff / 3600000);
  if (h < 1) return 'Just now';
  if (h < 48) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}

export type DebateHeroSwipeCardProps = {
  summary: DebateSummary;
  isLoggedIn: boolean;
  token: string | null;
  /** Navigate to full debate detail */
  onOpen: () => void;
  /** After a successful card vote (invalidate feed, etc.) */
  onVoteSuccess: () => void;
  buildPlaceholderMatch: (summary: DebateSummary) => Match;
};

export default function DebateHeroSwipeCard({
  summary,
  isLoggedIn,
  token,
  onOpen,
  onVoteSuccess,
  buildPlaceholderMatch,
}: DebateHeroSwipeCardProps) {
  const translateX = useSharedValue(0);
  const overlayDir = useSharedValue(0);

  const debateQuery = useQuery({
    queryKey: ['debateHero', summary.id],
    queryFn: async (): Promise<DebateResponse | null> => {
      const d = await fetchDebateById(summary.id);
      return d;
    },
    enabled: Number.isFinite(summary.id) && summary.id > 0,
  });

  const debate = debateQuery.data;
  const firstCard = debate?.cards?.[0];
  const canSwipe = !!debate?.id && firstCard?.id != null && !debateQuery.isLoading;

  const headline = summary.headline.toUpperCase();
  const votes = summary.analytics?.total_votes ?? 0;

  const hasSource =
    !!summary.source_headline?.trim() ||
    !!summary.source_url?.trim() ||
    !!summary.source_published_at?.trim();
  const sourceLabel = summary.source_headline?.trim()
    ? summary.source_headline
    : summary.source_url?.trim() ?? '';

  const openAuthForSwipe = useCallback(() => {
    if (!debate) return;
    const match = buildPlaceholderMatch(summary);
    rootNavigate('Login', {
      returnToDebate: {
        match,
        debate,
        pendingAction: 'swipe',
      },
    });
  }, [debate, summary, buildPlaceholderMatch]);

  const [voteBusy, setVoteBusy] = useState(false);

  const performVote = useCallback(
    async (voteType: 'upvote' | 'downvote') => {
      if (!debate?.id || firstCard?.id == null) return;
      if (!isLoggedIn || !token) {
        openAuthForSwipe();
        return;
      }
      if (voteBusy) return;
      setVoteBusy(true);
      try {
        const counts = await setCardVote(token, debate.id, firstCard.id, voteType);
        if (counts) onVoteSuccess();
      } finally {
        setVoteBusy(false);
      }
    },
    [
      debate?.id,
      firstCard?.id,
      isLoggedIn,
      token,
      voteBusy,
      onVoteSuccess,
      openAuthForSwipe,
    ],
  );

  const handlePanEnd = useCallback(
    (dx: number, dy: number) => {
      if (Math.abs(dx) < 18 && Math.abs(dy) < 18) {
        onOpen();
        return;
      }
      if (!canSwipe || voteBusy) return;
      if (dx > SWIPE_THRESHOLD) {
        if (!isLoggedIn || !token) openAuthForSwipe();
        else void performVote('upvote');
      } else if (dx < -SWIPE_THRESHOLD) {
        if (!isLoggedIn || !token) openAuthForSwipe();
        else void performVote('downvote');
      }
    },
    [
      canSwipe,
      voteBusy,
      onOpen,
      isLoggedIn,
      token,
      openAuthForSwipe,
      performVote,
    ],
  );

  const pan = useMemo(
    () =>
      Gesture.Pan()
        .enabled(!voteBusy)
        .activeOffsetX([-12, 12])
        .failOffsetY([-16, 16])
        .onUpdate(e => {
          translateX.value = e.translationX;
          if (e.translationX > 24) {
            overlayDir.value = 1;
          } else if (e.translationX < -24) {
            overlayDir.value = -1;
          } else {
            overlayDir.value = 0;
          }
        })
        .onEnd(e => {
          runOnJS(handlePanEnd)(e.translationX, e.translationY);
          translateX.value = withSpring(0);
          overlayDir.value = 0;
        }),
    [voteBusy, translateX, overlayDir, handlePanEnd],
  );

  const cardStyle = useAnimatedStyle(() => ({
    transform: [
      {translateX: translateX.value},
      {rotate: `${translateX.value / 25}deg`},
    ],
  }));

  const overlayYesStyle = useAnimatedStyle(() => ({
    opacity: overlayDir.value === 1 ? 0.95 : 0,
  }));
  const overlayNoStyle = useAnimatedStyle(() => ({
    opacity: overlayDir.value === -1 ? 0.95 : 0,
  }));

  return (
    <View style={styles.heroOuter}>
      <GestureDetector gesture={pan}>
        <Animated.View style={[styles.heroClip, cardStyle]}>
          <Image
            source={{uri: HERO_IMAGE_URI}}
            style={styles.heroImage}
            resizeMode="cover"
          />
          <LinearGradient
            colors={['rgba(11,14,20,0.15)', 'rgba(11,14,20,0.85)', BG]}
            locations={[0, 0.45, 1]}
            style={StyleSheet.absoluteFill}
          />
          <Animated.View
            style={[styles.swipeOverlay, styles.swipeOverlayYes, overlayYesStyle]}
            pointerEvents="none">
            <Ionicons name="thumbs-up" size={56} color={TEXT} />
          </Animated.View>
          <Animated.View
            style={[styles.swipeOverlay, styles.swipeOverlayNo, overlayNoStyle]}
            pointerEvents="none">
            <Ionicons name="thumbs-down" size={56} color={TEXT} />
          </Animated.View>
          <View style={styles.heroInner}>
            <View style={styles.pill}>
              <Text style={styles.pillText}>
                {debatePillLabel(summary.debate_type)}
              </Text>
            </View>
            <Text style={styles.heroHeadline}>{headline}</Text>
            {hasSource ? (
              <Text
                style={styles.heroSource}
                numberOfLines={2}
                onPress={() =>
                  summary.source_url
                    ? Linking.openURL(summary.source_url).catch(() => {})
                    : undefined
                }>
                {sourceLabel}
              </Text>
            ) : null}
            <View style={styles.heroStats}>
              <View style={styles.statItem}>
                <Ionicons name="time-outline" size={16} color={MUTED} />
                <Text style={styles.statText}>
                  {relativeTimeLabel(summary.created_at)}
                </Text>
              </View>
              <View style={styles.statItem}>
                <Ionicons name="people-outline" size={16} color={MUTED} />
                <Text style={styles.statText}>{formatVoteCount(votes)} voted</Text>
              </View>
            </View>
            {debateQuery.isLoading ? (
              <View style={styles.loadingRow}>
                <ActivityIndicator color={LIME} />
                <Text style={styles.loadingText}>Loading card…</Text>
              </View>
            ) : null}
            {!debateQuery.isLoading && !firstCard?.id ? (
              <Text style={styles.warnText}>No swipe card for this debate.</Text>
            ) : null}
            <View style={styles.swipeRow}>
              <View style={styles.swipeBtnRed} pointerEvents="none">
                <Ionicons name="close" size={22} color={TEXT} />
              </View>
              <View style={styles.swipeHint}>
                <View style={styles.swipeLine} />
                <Text style={styles.swipeHintText}>SWIPE TO VOTE</Text>
              </View>
              <View style={styles.swipeBtnLime} pointerEvents="none">
                <Ionicons name="checkmark" size={22} color="#0B0E14" />
              </View>
            </View>
            <View style={styles.swipeLabels}>
              <Text style={styles.disagreeLabel}>DISAGREE</Text>
              <Text style={styles.agreeLabel}>AGREE</Text>
            </View>
          </View>
        </Animated.View>
      </GestureDetector>
    </View>
  );
}

const styles = StyleSheet.create({
  heroOuter: {
    marginHorizontal: 16,
    marginBottom: 12,
    borderRadius: 16,
    overflow: 'hidden',
  },
  heroClip: {
    borderRadius: 16,
    overflow: 'hidden',
    minHeight: 320,
    backgroundColor: CARD,
  },
  heroImage: {
    ...StyleSheet.absoluteFillObject,
    width: '100%',
    height: '100%',
  },
  heroInner: {
    padding: 16,
    paddingTop: 20,
    minHeight: 300,
    justifyContent: 'flex-end',
  },
  pill: {
    alignSelf: 'flex-start',
    backgroundColor: LIME,
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 4,
    marginBottom: 12,
  },
  pillText: {
    fontSize: 10,
    fontWeight: '900',
    color: BG,
    letterSpacing: 0.8,
  },
  heroHeadline: {
    fontSize: 20,
    fontWeight: '900',
    color: TEXT,
    lineHeight: 26,
    letterSpacing: 0.3,
  },
  heroSource: {
    marginTop: 8,
    fontSize: 12,
    color: LIME,
    opacity: 0.92,
    lineHeight: 17,
  },
  heroStats: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 16,
    marginTop: 12,
  },
  statItem: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  statText: {
    fontSize: 13,
    color: MUTED,
    fontWeight: '600',
  },
  loadingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    marginTop: 12,
  },
  loadingText: {
    fontSize: 13,
    color: MUTED,
  },
  warnText: {
    marginTop: 8,
    fontSize: 12,
    color: MUTED,
  },
  swipeRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: 20,
    paddingHorizontal: 4,
  },
  swipeBtnRed: {
    width: 48,
    height: 48,
    borderRadius: 8,
    backgroundColor: RED_X,
    alignItems: 'center',
    justifyContent: 'center',
  },
  swipeBtnLime: {
    width: 48,
    height: 48,
    borderRadius: 8,
    backgroundColor: LIME,
    alignItems: 'center',
    justifyContent: 'center',
  },
  swipeHint: {
    flex: 1,
    alignItems: 'center',
    paddingHorizontal: 8,
  },
  swipeLine: {
    width: 56,
    height: 2,
    backgroundColor: MUTED,
    opacity: 0.5,
    marginBottom: 6,
  },
  swipeHintText: {
    fontSize: 9,
    fontWeight: '700',
    color: MUTED,
    letterSpacing: 1,
  },
  swipeLabels: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: 4,
    marginTop: 6,
    marginBottom: 4,
  },
  disagreeLabel: {
    fontSize: 11,
    fontWeight: '800',
    color: RED_X,
    width: 72,
    textAlign: 'center',
  },
  agreeLabel: {
    fontSize: 11,
    fontWeight: '800',
    color: LIME,
    width: 72,
    textAlign: 'center',
  },
  swipeOverlay: {
    ...StyleSheet.absoluteFillObject,
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 4,
  },
  swipeOverlayYes: {
    backgroundColor: 'rgba(198,255,0,0.35)',
  },
  swipeOverlayNo: {
    backgroundColor: 'rgba(255,59,48,0.4)',
  },
});
