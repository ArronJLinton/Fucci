import React, {useState, useEffect, useCallback, useRef} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ActivityIndicator,
  TouchableOpacity,
} from 'react-native';
import {Ionicons} from '@expo/vector-icons';
import Animated from 'react-native-reanimated';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import type {RootStackParamList} from '../types/navigation';
import type {Match} from '../types/match';
import type {DebateResponse, DebateType} from '../types/debate';
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

const FINISHED_STATUSES = ['FT', 'AET', 'PEN', 'FT_PEN', 'AET_PEN'];

function getDefaultDebateType(match: Match): DebateType {
  const short = match?.fixture?.status?.short ?? '';
  return FINISHED_STATUSES.includes(short) ? 'post_match' : 'pre_match';
}

function agreePercentFromDebate(d: DebateResponse): {
  agreePct: number;
  hasVotes: boolean;
} {
  let agree = 0;
  let disagree = 0;
  for (const c of d.cards ?? []) {
    const u = c.vote_counts?.upvotes ?? 0;
    if (c.stance === 'agree') agree += u;
    if (c.stance === 'disagree') disagree += u;
  }
  const t = agree + disagree;
  if (t < 1) {
    return {agreePct: 50, hasVotes: false};
  }
  return {agreePct: Math.round((agree / t) * 100), hasVotes: true};
}

function yesNoFromDebate(d: DebateResponse): {yes: number; no: number} {
  const t = d.card_vote_totals;
  if (t) {
    const ty = t.total_yes ?? 0;
    const tn = t.total_no ?? 0;
    const sum = ty + tn;
    if (sum >= 1) {
      return {
        yes: Math.round((ty / sum) * 100),
        no: Math.round((tn / sum) * 100),
      };
    }
  }
  let y = 0;
  let n = 0;
  for (const c of d.cards ?? []) {
    if (c.stance === 'agree') y += c.vote_counts?.upvotes ?? 0;
    if (c.stance === 'disagree') n += c.vote_counts?.upvotes ?? 0;
  }
  const sum = y + n;
  if (sum < 1) return {yes: 42, no: 58};
  return {
    yes: Math.round((y / sum) * 100),
    no: Math.round((n / sum) * 100),
  };
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

  const hot = debateList[0];
  const referee = debateList.length > 1 ? debateList[1] : null;
  const hotConsensus = agreePercentFromDebate(hot);
  const agreePct = hotConsensus.agreePct;
  const hasVotes = hotConsensus.hasVotes;
  const disagreePct = 100 - agreePct;
  const refYN = referee ? yesNoFromDebate(referee) : {yes: 50, no: 50};

  return (
    <View style={styles.container}>
      <Animated.ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
        onScroll={onScroll}
        scrollEventThrottle={1}
        bounces={false}
        overScrollMode="never"
        showsVerticalScrollIndicator={false}>
        <View style={styles.hotCard}>
          <View style={styles.hotHeader}>
            <View style={styles.hotPill}>
              <Text style={styles.hotPillText}>HOT TOPIC</Text>
            </View>
          </View>
          <Text style={styles.quoteText}>&ldquo;{hot.headline}&rdquo;</Text>
          <View style={styles.pollRow}>
            {hasVotes ? (
              <>
                <Text style={styles.agreeLabel}>AGREE ({agreePct}%)</Text>
                <Text style={styles.disagreeLabel}>
                  DISAGREE ({disagreePct}%)
                </Text>
              </>
            ) : (
              <Text style={styles.noVotesLabel}>NO VOTES YET</Text>
            )}
          </View>
          <View style={styles.barTrack}>
            {hasVotes ? (
              <View style={[styles.barFill, {width: `${agreePct}%`}]} />
            ) : (
              <View style={styles.barEmpty} />
            )}
          </View>
          <TouchableOpacity
            style={styles.joinCta}
            activeOpacity={0.9}
            onPress={() => openSingleDebate(hot, 0)}>
            <Text style={styles.joinCtaText}>JOIN THE CONVERSATION</Text>
            <Ionicons
              name="chatbubble-ellipses"
              size={18}
              color={MATCH_CENTER_BLACK}
            />
          </TouchableOpacity>
        </View>

        {referee ? (
          <View style={styles.refCard}>
            <View style={styles.refHeader}>
              <View style={styles.refPill}>
                <Text style={styles.refPillText}>REFEREE WATCH</Text>
              </View>
            </View>
            <Text style={styles.refQuestion}>{referee.headline}</Text>
            <View style={styles.refButtons}>
              <View style={styles.refBtn}>
                <Text style={styles.refBtnLabel}>YES</Text>
                <Text style={styles.refBtnPct}>{refYN.yes}%</Text>
              </View>
              <View style={styles.refBtn}>
                <Text style={styles.refBtnLabel}>NO</Text>
                <Text style={styles.refBtnPct}>{refYN.no}%</Text>
              </View>
            </View>
            <TouchableOpacity
              style={styles.refTap}
              activeOpacity={0.85}
              onPress={() => openSingleDebate(referee, 0)}>
              <Text style={styles.refTapText}>Open debate</Text>
              <Ionicons
                name="chevron-forward"
                size={18}
                color={MATCH_CENTER_LIME}
              />
            </TouchableOpacity>
          </View>
        ) : null}

        <View style={styles.gridRow}>
          <TouchableOpacity
            style={styles.gridCard}
            activeOpacity={0.9}
            onPress={() => openSingleDebate(hot, 0)}>
            <Ionicons name="star" size={22} color={MATCH_CENTER_CYAN} />
            <Text style={styles.gridTitle}>PLAYER OF THE MATCH</Text>
            <Text style={styles.gridCta}>Vote Now</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={styles.gridCard}
            activeOpacity={0.9}
            onPress={() => openSingleDebate(hot, 0)}>
            <View style={styles.gridIconWrap}>
              <Ionicons name="bar-chart" size={22} color={MATCH_CENTER_LIME} />
            </View>
            <Text style={styles.gridTitle}>TOP DEBATERS</Text>
            <Text style={styles.gridSub}>+45 pts today</Text>
          </TouchableOpacity>
        </View>
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
  hotCard: {
    backgroundColor: MATCH_CENTER_CARD,
    borderRadius: 12,
    padding: 16,
    marginBottom: 14,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
  },
  hotHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 12,
    flexWrap: 'wrap',
    gap: 8,
  },
  hotPill: {
    backgroundColor: MATCH_CENTER_CYAN,
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 6,
  },
  hotPillText: {
    fontSize: 10,
    fontWeight: '900',
    letterSpacing: 0.8,
    color: MATCH_CENTER_BLACK,
  },
  joinedMuted: {
    fontSize: 12,
    color: MATCH_CENTER_MUTED,
    fontWeight: '600',
  },
  quoteText: {
    fontSize: 16,
    fontWeight: '800',
    color: MATCH_CENTER_TEXT,
    lineHeight: 22,
    marginBottom: 14,
  },
  pollRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  agreeLabel: {
    fontSize: 12,
    fontWeight: '800',
    color: MATCH_CENTER_LIME,
    letterSpacing: 0.3,
  },
  disagreeLabel: {
    fontSize: 12,
    fontWeight: '800',
    color: MATCH_CENTER_MUTED,
    letterSpacing: 0.3,
  },
  noVotesLabel: {
    fontSize: 12,
    fontWeight: '800',
    color: MATCH_CENTER_MUTED,
    letterSpacing: 0.4,
  },
  barTrack: {
    height: 8,
    borderRadius: 4,
    backgroundColor: 'rgba(255,255,255,0.08)',
    overflow: 'hidden',
    marginBottom: 16,
  },
  barFill: {
    height: '100%',
    backgroundColor: MATCH_CENTER_LIME,
    borderRadius: 4,
  },
  barEmpty: {
    height: '100%',
    width: '100%',
    backgroundColor: 'rgba(255,255,255,0.04)',
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
  refCard: {
    backgroundColor: MATCH_CENTER_CARD,
    borderRadius: 12,
    padding: 16,
    marginBottom: 14,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
  },
  refHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 12,
    flexWrap: 'wrap',
    gap: 8,
  },
  refPill: {
    backgroundColor: 'rgba(255,255,255,0.1)',
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 6,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.12)',
  },
  refPillText: {
    fontSize: 10,
    fontWeight: '800',
    letterSpacing: 0.6,
    color: MATCH_CENTER_MUTED,
  },
  refQuestion: {
    fontSize: 15,
    fontWeight: '800',
    color: MATCH_CENTER_TEXT,
    lineHeight: 21,
    marginBottom: 14,
  },
  refButtons: {
    flexDirection: 'row',
    gap: 10,
  },
  refBtn: {
    flex: 1,
    backgroundColor: 'rgba(255,255,255,0.06)',
    borderRadius: 10,
    paddingVertical: 12,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.08)',
  },
  refBtnLabel: {
    fontSize: 13,
    fontWeight: '900',
    color: MATCH_CENTER_TEXT,
    letterSpacing: 0.5,
  },
  refBtnPct: {
    fontSize: 12,
    fontWeight: '700',
    color: MATCH_CENTER_MUTED,
    marginTop: 4,
  },
  refTap: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 12,
    gap: 4,
  },
  refTapText: {
    fontSize: 13,
    fontWeight: '700',
    color: MATCH_CENTER_MUTED,
  },
  gridRow: {
    flexDirection: 'row',
    gap: 10,
  },
  gridCard: {
    flex: 1,
    backgroundColor: MATCH_CENTER_CARD,
    borderRadius: 12,
    padding: 14,
    minHeight: 120,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
    justifyContent: 'space-between',
  },
  gridIconWrap: {
    marginBottom: 4,
  },
  gridTitle: {
    fontSize: 11,
    fontWeight: '900',
    letterSpacing: 0.5,
    color: MATCH_CENTER_TEXT,
    marginTop: 6,
  },
  gridCta: {
    fontSize: 12,
    fontWeight: '800',
    color: MATCH_CENTER_CYAN,
    marginTop: 8,
  },
  gridSub: {
    fontSize: 12,
    fontWeight: '700',
    color: MATCH_CENTER_LIME,
    marginTop: 8,
  },
});

export default DebateScreen;
