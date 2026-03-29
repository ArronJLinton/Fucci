import React, {useCallback, useEffect, useMemo, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  Image,
  Linking,
  Dimensions,
  TouchableOpacity,
} from 'react-native';
import {LinearGradient} from 'expo-linear-gradient';
import {Gesture, GestureDetector} from 'react-native-gesture-handler';
import Animated, {
  runOnJS,
  runOnUI,
  useAnimatedStyle,
  useSharedValue,
  withSpring,
  withTiming,
} from 'react-native-reanimated';
import {Ionicons} from '@expo/vector-icons';
import {useQuery} from '@tanstack/react-query';
import type {DebateSummary, DebateResponse, DebateCard} from '../types/debate';
import type {Match} from '../types/match';
import {fetchDebateById, setCardVote} from '../services/debate';
import {rootNavigate} from '../navigation/rootNavigation';

const BG = '#0B0E14';
const LIME = '#C6FF00';
const CARD = '#1A1F2E';
const TEXT = '#FFFFFF';
const MUTED = '#8B92A5';
const RED_X = '#FF3B30';
const HERO_IMAGE = require('../../assets/debate-hero.jpg');
const HERO_IMAGE_URI = Image.resolveAssetSource(HERO_IMAGE).uri;

const SWIPE_THRESHOLD = 80;
const TAP_MAX = 18;
/** Fly card fully past the clip; scales with screen width. */
const OFFSCREEN_X = Dimensions.get('window').width * 1.35;

/** Binary hero: need both stance cards so swipe-right/upvote-agree vs swipe-left/upvote-disagree match feed `binary_consensus`. */
function pickBinaryStanceCards(cards: DebateCard[] | undefined): {
  agree?: DebateCard;
  disagree?: DebateCard;
} {
  if (!cards?.length) return {};
  return {
    agree: cards.find(c => c.stance === 'agree'),
    disagree: cards.find(c => c.stance === 'disagree'),
  };
}

function debatePillLabel(debateType: string): string {
  return debateType === 'post_match' ? 'CONTROVERSY' : 'PRE-MATCH';
}

function formatVoteCount(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
  return String(n);
}

function relativeTimeLabel(iso: string): string {
  const raw = iso?.trim() ?? '';
  if (!raw) return '';
  const t = new Date(raw).getTime();
  if (Number.isNaN(t)) return '';
  const diff = Date.now() - t;
  const h = Math.floor(diff / 3600000);
  if (h < 1) return 'Just now';
  if (h < 48) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}

/** Fired once per pan end (JS thread) — use for screen-level logging / analytics. */
export type DebateHeroPanResult =
  | {summaryId: number; kind: 'tap_open'; dx: number; dy: number}
  | {
      summaryId: number;
      kind: 'ignored';
      reason: 'not_ready_or_no_card' | 'voteBusy' | 'auth_required';
      dx: number;
      dy: number;
    }
  | {
      summaryId: number;
      kind: 'swipe_right';
      outcome: 'agree';
      dx: number;
      dy: number;
    }
  | {
      summaryId: number;
      kind: 'swipe_left';
      outcome: 'disagree';
      dx: number;
      dy: number;
    }
  | {
      summaryId: number;
      kind: 'swipe_incomplete';
      dx: number;
      dy: number;
      absDx: number;
      threshold: number;
    };
/** Passed when POST vote succeeded and counts were returned (swipe vote persisted). */
export type DebateHeroVoteSuccessDetail = {
  debateId: number;
  cardId: number;
  voteType: 'upvote' | 'downvote';
};

export type DebateHeroSwipeCardProps = {
  summary: DebateSummary;
  isLoggedIn: boolean;
  token: string | null;
  /** Navigate to full debate detail */
  onOpen: () => void;
  /** After a successful swipe vote (server accepted vote); use detail to confirm persistence. */
  onVoteSuccess: (detail: DebateHeroVoteSuccessDetail) => void;
  buildPlaceholderMatch: (summary: DebateSummary) => Match;
  /** Optional: parent can log each resolved pan (e.g. MainDebatesScreen). */
  onPanResolved?: (result: DebateHeroPanResult) => void;
};

export default function DebateHeroSwipeCard({
  summary,
  isLoggedIn,
  token,
  onOpen,
  onVoteSuccess,
  buildPlaceholderMatch,
  onPanResolved,
}: DebateHeroSwipeCardProps) {
  const translateX = useSharedValue(0);
  const overlayDir = useSharedValue(0);
  /** Mirrored for pan worklets — 1 only when swipe voting is allowed (logged in + binary cards). */
  const voteEnabledSV = useSharedValue(0);
  const voteBusySV = useSharedValue(0);

  const debateQuery = useQuery({
    queryKey: ['debateHero', summary.id],
    queryFn: async (): Promise<DebateResponse | null> => {
      const d = await fetchDebateById(summary.id);
      return d;
    },
    enabled: Number.isFinite(summary.id) && summary.id > 0,
    staleTime: 5 * 60 * 1000,
  });

  const debate = debateQuery.data;
  const {agree: agreeCard, disagree: disagreeCard} = useMemo(
    () => pickBinaryStanceCards(debate?.cards),
    [debate?.cards],
  );
  /** Debate loaded with both stance cards — UI can show vote affordances (guest sees sign-in). */
  const binaryVoteUiReady =
    !!debate?.id &&
    agreeCard?.id != null &&
    disagreeCard?.id != null &&
    !debateQuery.isLoading;
  /** Logged-in only: swipe-to-vote. */
  const canSwipeVote = binaryVoteUiReady && isLoggedIn && !!token;
  const authRequiredForVote = binaryVoteUiReady && (!isLoggedIn || !token);

  const [voteBusy, setVoteBusy] = useState(false);

  useEffect(() => {
    voteEnabledSV.value = canSwipeVote ? 1 : 0;
  }, [canSwipeVote, voteEnabledSV]);

  useEffect(() => {
    voteBusySV.value = voteBusy ? 1 : 0;
  }, [voteBusy, voteBusySV]);

  useEffect(() => {
    translateX.value = 0;
    overlayDir.value = 0;
  }, [summary.id]);

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
    if (!debate) {
      return;
    }
    const match = buildPlaceholderMatch(summary);
    rootNavigate('Login', {
      returnToDebate: {
        match,
        debate,
        pendingAction: 'swipe',
      },
    });
  }, [debate, summary, buildPlaceholderMatch]);

  const springHeroCardBack = useCallback(() => {
    runOnUI(() => {
      'worklet';
      translateX.value = withSpring(0);
    })();
  }, [translateX]);

  const flyHeroCardOff = useCallback((direction: 'left' | 'right') => {
    const target = direction === 'right' ? OFFSCREEN_X : -OFFSCREEN_X;
    runOnUI(() => {
      'worklet';
      translateX.value = withTiming(target, {duration: 280});
    })();
  }, [translateX]);

  /** Swipe right → upvote agree card; swipe left → upvote disagree card (matches API binary_consensus). */
  const performBinarySwipeVote = useCallback(
    async (side: 'agree' | 'disagree'): Promise<boolean> => {
      const card = side === 'agree' ? agreeCard : disagreeCard;
      if (!debate?.id || card?.id == null) {
        return false;
      }
      if (!isLoggedIn || !token) {
        return false;
      }
      if (voteBusy) {
        return false;
      }
      setVoteBusy(true);
      try {
        const counts = await setCardVote(token, debate.id, card.id, 'upvote');
        if (counts) {
          onVoteSuccess({
            debateId: debate.id,
            cardId: card.id,
            voteType: 'upvote',
          });
          return true;
        }
        return false;
      } catch {
        return false;
      } finally {
        setVoteBusy(false);
      }
    },
    [
      debate?.id,
      agreeCard?.id,
      disagreeCard?.id,
      isLoggedIn,
      token,
      voteBusy,
      onVoteSuccess,
    ],
  );

  const handlePanEnd = useCallback(
    (dx: number, dy: number) => {
      const sid = summary.id;
      if (Math.abs(dx) < TAP_MAX && Math.abs(dy) < TAP_MAX) {
        onPanResolved?.({summaryId: sid, kind: 'tap_open', dx, dy});
        onOpen();
        return;
      }
      if (voteBusy) {
        onPanResolved?.({summaryId: sid, kind: 'ignored', reason: 'voteBusy', dx, dy});
        return;
      }
      if (!canSwipeVote) {
        const reason = authRequiredForVote ? 'auth_required' : 'not_ready_or_no_card';
        onPanResolved?.({summaryId: sid, kind: 'ignored', reason, dx, dy});
        springHeroCardBack();
        return;
      }
      if (dx > SWIPE_THRESHOLD) {
        onPanResolved?.({summaryId: sid, kind: 'swipe_right', outcome: 'agree', dx, dy});
        void (async () => {
          const ok = await performBinarySwipeVote('agree');
          if (ok) {
            flyHeroCardOff('right');
          } else {
            springHeroCardBack();
          }
        })();
      } else if (dx < -SWIPE_THRESHOLD) {
        onPanResolved?.({summaryId: sid, kind: 'swipe_left', outcome: 'disagree', dx, dy});
        void (async () => {
          const ok = await performBinarySwipeVote('disagree');
          if (ok) {
            flyHeroCardOff('left');
          } else {
            springHeroCardBack();
          }
        })();
      } else {
        const absDx = Math.abs(dx);
        onPanResolved?.({
          summaryId: sid,
          kind: 'swipe_incomplete',
          dx,
          dy,
          absDx,
          threshold: SWIPE_THRESHOLD,
        });
      }
    },
    [
      summary.id,
      canSwipeVote,
      authRequiredForVote,
      voteBusy,
      onOpen,
      onPanResolved,
      performBinarySwipeVote,
      springHeroCardBack,
      flyHeroCardOff,
    ],
  );

  const pan = useMemo(
    () =>
      Gesture.Pan()
        .enabled(!voteBusy)
        .activeOffsetX([-12, 12])
        .failOffsetY([-16, 16])
        .onUpdate(e => {
          if (voteEnabledSV.value === 0) {
            translateX.value = 0;
            overlayDir.value = 0;
            return;
          }
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
          const dx = e.translationX;
          const dy = e.translationY;
          overlayDir.value = 0;

          const isTap =
            Math.abs(dx) < TAP_MAX && Math.abs(dy) < TAP_MAX;
          if (isTap) {
            translateX.value = withSpring(0);
            runOnJS(handlePanEnd)(dx, dy);
            return;
          }

          const ready = voteEnabledSV.value === 1;
          const busy = voteBusySV.value === 1;
          if (!ready || busy) {
            translateX.value = withSpring(0);
            runOnJS(handlePanEnd)(dx, dy);
            return;
          }

          if (dx > SWIPE_THRESHOLD) {
            runOnJS(handlePanEnd)(dx, dy);
          } else if (dx < -SWIPE_THRESHOLD) {
            runOnJS(handlePanEnd)(dx, dy);
          } else {
            translateX.value = withSpring(0);
            runOnJS(handlePanEnd)(dx, dy);
          }
        }),
    [voteBusy, translateX, overlayDir, handlePanEnd, voteEnabledSV, voteBusySV],
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

  const heroA11yLabel = `Featured debate: ${summary.headline}`;
  const heroA11yHint = authRequiredForVote
    ? 'Sign in to vote. Short tap opens the full debate.'
    : 'Swipe right to agree, left to disagree. Short tap opens the full debate.';

  return (
    <View
      style={styles.heroOuter}
      accessible
      accessibilityLabel={heroA11yLabel}
      accessibilityHint={heroA11yHint}>
      <GestureDetector gesture={pan}>
        <Animated.View
          style={[styles.heroClip, cardStyle]}
          accessibilityElementsHidden>
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
            {!debateQuery.isLoading &&
            debate &&
            (!agreeCard?.id || !disagreeCard?.id) ? (
              <Text style={styles.warnText}>
                This debate needs agree and disagree cards to swipe-vote.
              </Text>
            ) : null}
            {authRequiredForVote ? (
              <View style={styles.guestVoteCta} accessibilityRole="text">
                <Text style={styles.guestVoteText}>
                  Sign in to participate in debates and cast your vote.
                </Text>
                <TouchableOpacity
                  onPress={openAuthForSwipe}
                  style={styles.guestSignInBtn}
                  accessibilityRole="button"
                  accessibilityLabel="Sign in to vote">
                  <Text style={styles.guestSignInBtnText}>Sign in</Text>
                  <Ionicons name="chevron-forward" size={18} color={BG} />
                </TouchableOpacity>
              </View>
            ) : canSwipeVote ? (
              <>
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
              </>
            ) : null}
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
  warnText: {
    marginTop: 8,
    fontSize: 12,
    color: MUTED,
  },
  guestVoteCta: {
    marginTop: 20,
    paddingVertical: 12,
    paddingHorizontal: 4,
    gap: 12,
  },
  guestVoteText: {
    fontSize: 13,
    color: MUTED,
    fontWeight: '600',
    lineHeight: 19,
  },
  guestSignInBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    alignSelf: 'flex-start',
    gap: 4,
    backgroundColor: LIME,
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderRadius: 8,
  },
  guestSignInBtnText: {
    fontSize: 14,
    fontWeight: '800',
    color: BG,
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
