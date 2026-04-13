import React, {useState, useEffect, useCallback, useRef, useMemo} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ActivityIndicator,
  TouchableOpacity,
} from 'react-native';
import {Ionicons} from '@expo/vector-icons';
import Animated, {
  Easing,
  useAnimatedStyle,
  useSharedValue,
  withRepeat,
  withSequence,
  withTiming,
} from 'react-native-reanimated';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import type {RootStackParamList} from '../types/navigation';
import type {Match} from '../types/match';
import type {DebateCard, DebateResponse, DebateType} from '../types/debate';
import {
  fetchDebatesByMatch,
  fetchDebateById,
  generateDebateSet,
} from '../services/api';
import {
  useMatchDetailsScroll,
  type MatchDetailsScrollHandler,
} from '../context/MatchDetailsScrollContext';
import {
  MATCH_CENTER_BG,
  MATCH_CENTER_BLACK,
  MATCH_CENTER_LIME,
  MATCH_CENTER_CYAN,
  MATCH_CENTER_CARD,
  MATCH_CENTER_MUTED,
  MATCH_CENTER_TEXT,
} from '../constants/matchCenterUi';

/** Same semantics as SingleDebateScreen `binaryPulseSideTotals` (read-only on this screen). */
function binaryPulseSideTotals(
  binaryCards: DebateCard[],
  localCounts: Record<number, {upvotes: number; downvotes: number}>,
): {agreeVotes: number; disagreeVotes: number} {
  let agreeVotes = 0;
  let disagreeVotes = 0;
  for (const c of binaryCards) {
    if (c.id == null) continue;
    const counts = localCounts[c.id] ?? {
      upvotes: c.vote_counts?.upvotes ?? 0,
      downvotes: c.vote_counts?.downvotes ?? 0,
    };
    if (c.stance === 'agree') {
      agreeVotes += counts.upvotes;
      disagreeVotes += counts.downvotes;
    } else if (c.stance === 'disagree') {
      disagreeVotes += counts.upvotes + counts.downvotes;
    }
  }
  return {agreeVotes, disagreeVotes};
}

const PULSE_DISAGREE = '#FF3B30';

const MatchDebatePulse: React.FC<{cards: DebateCard[] | undefined}> = ({
  cards,
}) => {
  const binaryCards = useMemo(
    () =>
      (cards ?? []).filter(
        c => c.stance === 'agree' || c.stance === 'disagree',
      ),
    [cards],
  );

  const glow = useSharedValue(1);

  useEffect(() => {
    if (binaryCards.length === 0) {
      return;
    }
    glow.value = withRepeat(
      withSequence(
        withTiming(0.82, {
          duration: 1100,
          easing: Easing.inOut(Easing.ease),
        }),
        withTiming(1, {
          duration: 1100,
          easing: Easing.inOut(Easing.ease),
        }),
      ),
      -1,
      false,
    );
  }, [binaryCards.length, glow]);

  const pulseStyle = useAnimatedStyle(() => ({
    opacity: glow.value,
  }));

  const {agreeVotes, disagreeVotes} = useMemo(
    () => binaryPulseSideTotals(binaryCards, {}),
    [binaryCards],
  );

  if (binaryCards.length === 0) {
    return null;
  }

  const totalSide = agreeVotes + disagreeVotes;
  const agreePct =
    totalSide > 0 ? Math.round((agreeVotes / totalSide) * 100) : 0;
  const disagreePct = totalSide > 0 ? 100 - agreePct : 0;

  const agreeCard = binaryCards.find(c => c.stance === 'agree');
  const disagreeCard = binaryCards.find(c => c.stance === 'disagree');
  const agreeSubHead = agreeCard?.title?.trim() || 'Agree';
  const disagreeSubHead = disagreeCard?.title?.trim() || 'Disagree';

  return (
    <Animated.View style={[styles.pulseWrap, pulseStyle]}>
      <View style={styles.pulseSubHeadRow}>
        <Text
          style={styles.pulseSubHeadAgree}
          numberOfLines={2}>
          {agreeSubHead}
        </Text>
        <Text
          style={styles.pulseSubHeadDisagree}
          numberOfLines={2}>
          {disagreeSubHead}
        </Text>
      </View>
      <View style={styles.pulseMiniPctRow}>
        <Text style={styles.pulsePctSmallAgree}>{agreePct}%</Text>
        <Text style={styles.pulsePctSmallDisagree}>{disagreePct}%</Text>
      </View>
      <View style={styles.pulseBarTrack}>
        <View style={[styles.pulseSegAgree, {width: `${agreePct}%`}]} />
        <View
          style={[styles.pulseSegDisagree, {width: `${disagreePct}%`}]}
        />
      </View>
    </Animated.View>
  );
};

const FINISHED_STATUSES = ['FT', 'AET', 'PEN', 'FT_PEN', 'AET_PEN'];

/** Labels for the up-to-three match debates returned by generate-set / DB. */
const MATCH_DEBATE_PILLS = ['HOT TOPIC', 'REFEREE WATCH', 'KEY TALKING POINT'] as const;

function getDefaultDebateType(match: Match): DebateType {
  const short = match?.fixture?.status?.short ?? '';
  return FINISHED_STATUSES.includes(short) ? 'post_match' : 'pre_match';
}

interface DebateScreenProps {
  match: Match;
  stackNavigation?: NativeStackNavigationProp<RootStackParamList>;
  matchScrollHandler?: MatchDetailsScrollHandler;
}

const PAGE = 16;

const DebateScreen: React.FC<DebateScreenProps> = ({
  match,
  stackNavigation,
  matchScrollHandler,
}) => {
  const stackNav = stackNavigation ?? null;
  const matchScroll = useMatchDetailsScroll();
  const onScroll = matchScrollHandler ?? matchScroll?.scrollHandler;

  const [debateList, setDebateList] = useState<DebateResponse[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [debateType, setDebateType] = useState<DebateType>(() =>
    getDefaultDebateType(match),
  );

  const loadCancelledRef = useRef(false);

  const openSingleDebate = (debate: DebateResponse, selectedCardIndex = 0) => {
    if (!stackNav) return;
    stackNav.navigate('SingleDebate', {match, debate, selectedCardIndex});
  };

  const loadDebateForType = useCallback(
    async (type: DebateType) => {
      if (!match?.fixture?.id) {
        setError('Invalid match: missing fixture ID');
        setIsLoading(false);
        return;
      }
      setError(null);
      setIsLoading(true);
      setDebateList([]);
      const matchId = match.fixture.id;
      const POLL_INTERVAL_MS = 3000;
      const POLL_TIMEOUT_MS = 60000;

      const cancelled = () => loadCancelledRef.current;

      try {
        let list = await fetchDebatesByMatch(matchId, type);
        if (cancelled()) return;
        if (list.length > 0) {
          const fullDebates: DebateResponse[] = [];
          for (const item of list) {
            const full = await fetchDebateById(item.id);
            if (cancelled()) return;
            if (full) fullDebates.push(full);
          }
          if (cancelled()) return;
          setDebateList(fullDebates);
          return;
        }

        setIsGenerating(true);
        const setResult = await generateDebateSet(matchId, type, 3);
        if (cancelled()) return;
        if (setResult?.rateLimited) {
          setError('Rate limit reached. Try again later.');
          return;
        }
        if (setResult?.debates?.length) {
          setDebateList(setResult.debates);
          return;
        }
        if (setResult?.pending) {
          const deadline = Date.now() + POLL_TIMEOUT_MS;
          while (Date.now() < deadline) {
            if (cancelled()) break;
            await new Promise(r => setTimeout(r, POLL_INTERVAL_MS));
            if (cancelled()) break;
            list = await fetchDebatesByMatch(matchId, type);
            if (cancelled()) break;
            if (list.length > 0) {
              const fullDebates: DebateResponse[] = [];
              for (const item of list) {
                const full = await fetchDebateById(item.id);
                if (cancelled()) break;
                if (full) fullDebates.push(full);
              }
              if (cancelled()) break;
              setDebateList(fullDebates);
              return;
            }
          }
          if (cancelled()) return;
        }

        if (setResult !== null) {
          setError('No debates generated. Try again later.');
          return;
        }

        list = await fetchDebatesByMatch(matchId, type);
        if (cancelled()) return;
        if (list.length > 0) {
          const fullDebates: DebateResponse[] = [];
          for (const item of list) {
            const full = await fetchDebateById(item.id);
            if (cancelled()) return;
            if (full) fullDebates.push(full);
          }
          if (cancelled()) return;
          setDebateList(fullDebates);
        } else {
          setError('Could not load debates. Try again.');
        }
      } catch (err) {
        if (cancelled()) return;
        const msg =
          err instanceof Error ? err.message : 'Failed to load debate';
        setError(msg);
        setDebateList([]);
      } finally {
        if (!cancelled()) {
          setIsLoading(false);
          setIsGenerating(false);
        }
      }
    },
    [match?.fixture?.id],
  );

  useEffect(() => {
    loadCancelledRef.current = false;
    loadDebateForType(debateType);
    return () => {
      loadCancelledRef.current = true;
    };
  }, [debateType, loadDebateForType]);

  useEffect(() => {
    const defaultType = getDefaultDebateType(match);
    setDebateType(prev => (prev !== defaultType ? defaultType : prev));
  }, [match?.fixture?.id, match?.fixture?.status?.short]);

  const showLoading = isLoading || isGenerating;
  const loadingMessage = isGenerating
    ? 'Generating debate...'
    : 'Loading debate...';

  if (showLoading) {
    return (
      <View style={styles.centerContainer}>
        <ActivityIndicator size="large" color={MATCH_CENTER_LIME} />
        <Text style={styles.loadingText}>{loadingMessage}</Text>
      </View>
    );
  }

  if (error) {
    return (
      <View style={styles.centerContainer}>
        <Text style={styles.errorText}>{error}</Text>
      </View>
    );
  }

  if (!debateList.length) {
    return (
      <View style={styles.centerContainer}>
        <Text style={styles.noDataText}>No debates yet</Text>
        <Text style={styles.emptySubtext}>
          Debates for this match have not been generated yet.
        </Text>
      </View>
    );
  }

  const debatesToShow = debateList.slice(0, 3);

  return (
    <View style={styles.container}>
      <Animated.ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
        onScroll={onScroll}
        scrollEventThrottle={16}
        bounces={false}
        overScrollMode="never"
        showsVerticalScrollIndicator={false}>
        {debatesToShow.map((d, index) => {
          const pillLabel =
            MATCH_DEBATE_PILLS[index] ?? `DEBATE ${index + 1}`;
          const pillHot = index === 0;
          return (
            <View
              key={d.id ?? `debate-${index}`}
              style={styles.topicCard}>
              <View style={styles.topicHeader}>
                <View
                  style={[
                    styles.topicPill,
                    pillHot ? styles.topicPillHot : styles.topicPillMuted,
                  ]}>
                  <Text
                    style={[
                      styles.topicPillText,
                      pillHot ? styles.topicPillTextHot : styles.topicPillTextMuted,
                    ]}>
                    {pillLabel}
                  </Text>
                </View>
              </View>
              <Text style={styles.quoteText}>
                &ldquo;{d.headline}&rdquo;
              </Text>
              <MatchDebatePulse cards={d.cards} />
              <TouchableOpacity
                style={styles.joinCta}
                activeOpacity={0.9}
                onPress={() => openSingleDebate(d, 0)}
                accessibilityRole="button"
                accessibilityLabel="Join the conversation">
                <Text style={styles.joinCtaText}>JOIN THE CONVERSATION</Text>
                <Ionicons
                  name="chatbubble-ellipses"
                  size={18}
                  color={MATCH_CENTER_BLACK}
                />
              </TouchableOpacity>
            </View>
          );
        })}
      </Animated.ScrollView>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: MATCH_CENTER_BG,
  },
  scrollView: {
    flex: 1,
    backgroundColor: MATCH_CENTER_BG,
  },
  scrollContent: {
    paddingHorizontal: PAGE,
    paddingTop: 12,
    paddingBottom: 28,
  },
  centerContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: MATCH_CENTER_BG,
    paddingHorizontal: 24,
  },
  loadingText: {
    marginTop: 12,
    fontSize: 15,
    color: MATCH_CENTER_MUTED,
  },
  errorText: {
    fontSize: 15,
    color: '#ff6b6b',
    textAlign: 'center',
  },
  noDataText: {
    fontSize: 17,
    fontWeight: '800',
    color: MATCH_CENTER_TEXT,
    textAlign: 'center',
  },
  emptySubtext: {
    fontSize: 14,
    color: MATCH_CENTER_MUTED,
    marginTop: 8,
    textAlign: 'center',
  },
  topicCard: {
    backgroundColor: MATCH_CENTER_CARD,
    borderRadius: 12,
    padding: 16,
    marginBottom: 14,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
  },
  topicHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
  },
  topicPill: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 6,
  },
  topicPillHot: {
    backgroundColor: MATCH_CENTER_CYAN,
  },
  topicPillMuted: {
    backgroundColor: 'rgba(255,255,255,0.1)',
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.12)',
  },
  topicPillText: {
    fontSize: 10,
    fontWeight: '900',
    letterSpacing: 0.8,
  },
  topicPillTextHot: {
    color: MATCH_CENTER_BLACK,
  },
  topicPillTextMuted: {
    color: MATCH_CENTER_MUTED,
  },
  quoteText: {
    fontSize: 16,
    fontWeight: '800',
    color: MATCH_CENTER_TEXT,
    lineHeight: 22,
    marginBottom: 12,
  },
  pulseWrap: {
    marginBottom: 14,
  },
  pulseSubHeadRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    gap: 10,
    marginBottom: 6,
  },
  pulseSubHeadAgree: {
    flex: 1,
    fontSize: 11,
    fontWeight: '700',
    lineHeight: 14,
    color: MATCH_CENTER_LIME,
  },
  pulseSubHeadDisagree: {
    flex: 1,
    fontSize: 11,
    fontWeight: '700',
    lineHeight: 14,
    color: PULSE_DISAGREE,
    textAlign: 'right',
  },
  pulseMiniPctRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 6,
  },
  pulsePctSmallAgree: {
    fontSize: 11,
    fontWeight: '700',
    color: MATCH_CENTER_LIME,
  },
  pulsePctSmallDisagree: {
    fontSize: 11,
    fontWeight: '700',
    color: PULSE_DISAGREE,
  },
  pulseBarTrack: {
    flexDirection: 'row',
    height: 8,
    borderRadius: 999,
    overflow: 'hidden',
    backgroundColor: 'rgba(255,255,255,0.08)',
  },
  pulseSegAgree: {
    height: '100%',
    backgroundColor: MATCH_CENTER_LIME,
  },
  pulseSegDisagree: {
    height: '100%',
    backgroundColor: PULSE_DISAGREE,
  },
  joinCta: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 10,
    backgroundColor: MATCH_CENTER_LIME,
    paddingVertical: 14,
    borderRadius: 10,
  },
  joinCtaText: {
    fontSize: 13,
    fontWeight: '900',
    letterSpacing: 0.8,
    color: MATCH_CENTER_BLACK,
  },
});

export default DebateScreen;
