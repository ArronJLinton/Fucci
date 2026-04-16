import React, {useCallback, useEffect, useMemo, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  RefreshControl,
  ActivityIndicator,
  Linking,
  SectionList,
  StatusBar,
} from 'react-native';
import type {SectionListRenderItem} from 'react-native';
import {useQuery, useQueryClient} from '@tanstack/react-query';
import {useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {Ionicons} from '@expo/vector-icons';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import {useAuth} from '../context/AuthContext';
import {
  fetchDebatesPublicFeed,
  fetchDebatesFeed,
  fetchDebateById,
} from '../services/debate';
import type {DebateSummary, DebateResponse} from '../types/debate';
import type {Match} from '../types/match';
import type {DebatesStackParamList} from '../types/navigation';
import {userFacingApiMessage} from '../services/api';
import {rootNavigateToProfileAuth} from '../navigation/authNavigationActions';
import DebateHeroSwipeCard, {
  type DebateHeroVoteFailedDetail,
  type DebateHeroVoteSuccessDetail,
} from '../components/DebateHeroSwipeCard';

/** Design tokens — Velocity Strike–style dark + lime (009 spec references) */
const BG = '#0B0E14';
const LIME = '#C6FF00';
const CARD = '#1A1F2E';
const TEXT = '#FFFFFF';
const MUTED = '#8B92A5';
const RED = '#FF3B30';
const LIVE_DOT = '#3B82F6';

/** NEW DEBATES and MY ACTIVITY use `created_at` within this window (generation time). */
const FEED_MAX_AGE_MS = 6 * 24 * 60 * 60 * 1000;

/**
 * Go encodes `time.Time` as RFC3339Nano. `Date.parse` often returns NaN when fractional
 * seconds go past milliseconds (e.g. `.123456789Z`), so every row was filtered out.
 */
function parseFeedTimestampMs(iso: string | undefined): number | null {
  if (iso == null) {
    return null;
  }
  const s0 = String(iso).trim();
  if (!s0) {
    return null;
  }
  const tryParse = (s: string): number | null => {
    const n = Date.parse(s);
    return Number.isNaN(n) ? null : n;
  };
  let ms = tryParse(s0);
  if (ms != null) {
    return ms;
  }
  const withoutSubMsFrac = s0.replace(/(\.\d{3})\d+(?=[Zz]|[+-])/, '$1');
  ms = tryParse(withoutSubMsFrac);
  if (ms != null) {
    return ms;
  }
  const spaceToT = s0.includes('T') ? s0 : s0.replace(' ', 'T');
  return tryParse(spaceToT.replace(/(\.\d{3})\d+(?=[Zz]|[+-])/, '$1'));
}

function debateRowTimeMs(summary: DebateSummary): number | null {
  return (
    parseFeedTimestampMs(summary.created_at) ??
    parseFeedTimestampMs(summary.updated_at)
  );
}

function debateGeneratedWithinPastSixDays(
  summary: DebateSummary,
  nowMs: number,
): boolean {
  const rowMs = debateRowTimeMs(summary);
  if (rowMs == null) {
    return true;
  }
  return nowMs - rowMs <= FEED_MAX_AGE_MS;
}

type MainDebatesNavigation = NativeStackNavigationProp<
  DebatesStackParamList,
  'MainDebates'
>;

function buildPlaceholderMatch(summary: DebateSummary): Match {
  const raw = String(summary.match_id).replace(/\D/g, '');
  const fid = raw ? parseInt(raw, 10) : 0;
  const y = new Date().getFullYear();
  return {
    fixture: {
      id: fid,
      date: summary.created_at,
      status: {long: 'Scheduled', short: 'NS', elapsed: 0},
    },
    league: {id: 0, name: '', logo: '', season: y},
    teams: {
      home: {name: 'Home', logo: '', winner: null},
      away: {name: 'Away', logo: '', winner: null},
    },
    goals: {home: null, away: null},
  };
}

type UnifiedFeed =
  | {kind: 'public'; debates: DebateSummary[]}
  | {
      kind: 'auth';
      new_debates: DebateSummary[];
      voted_debates: DebateSummary[];
    };

/** Virtualized list rows — `kind` discriminates for SectionList renderItem / keyExtractor. */
type FeedRow =
  | {kind: 'hero'; summary: DebateSummary}
  | {kind: 'heroEmpty'}
  | {kind: 'guestCta'}
  | {kind: 'activityEmpty'}
  | {kind: 'activity'; summary: DebateSummary};

type MainDebatesSection = {
  key: 'new' | 'activity';
  data: FeedRow[];
};

/** Agree vs disagree % from feed binary tallies; null when missing or zero total. */
function consensusPercents(summary: DebateSummary): {
  agreePct: number;
  disagreePct: number;
} | null {
  const bc = summary.binary_consensus;
  if (!bc) return null;
  const a = bc.agree_upvotes;
  const d = bc.disagree_upvotes;
  const t = a + d;
  if (t <= 0) return null;
  const agreePct = Math.round((a / t) * 100);
  return {agreePct, disagreePct: 100 - agreePct};
}

/** After hero vote, bump feed consensus so MY ACTIVITY shows % before refetch completes. */
function withHeroVoteInBinaryConsensus(
  summary: DebateSummary,
  stance: 'agree' | 'disagree',
): DebateSummary {
  const prev = summary.binary_consensus ?? {
    agree_upvotes: 0,
    disagree_upvotes: 0,
  };
  return {
    ...summary,
    binary_consensus:
      stance === 'agree'
        ? {
            agree_upvotes: prev.agree_upvotes + 1,
            disagree_upvotes: prev.disagree_upvotes,
          }
        : {
            agree_upvotes: prev.agree_upvotes,
            disagree_upvotes: prev.disagree_upvotes + 1,
          },
  };
}

const MainDebatesScreen = () => {
  const navigation = useNavigation<MainDebatesNavigation>();
  const queryClient = useQueryClient();
  const insets = useSafeAreaInsets();
  const {isLoggedIn, token, isReady, user} = useAuth();

  const [voteErrorToast, setVoteErrorToast] = useState<string | null>(null);
  const [isPullRefreshing, setIsPullRefreshing] = useState(false);
  useEffect(() => {
    if (!voteErrorToast) {
      return;
    }
    const t = setTimeout(() => setVoteErrorToast(null), 4500);
    return () => clearTimeout(t);
  }, [voteErrorToast]);

  /** Segment feed cache per account; avoids showing another user's feed after switch. */
  const mainDebatesFeedQueryKey = useMemo(
    (): readonly ['mainDebatesFeed', number | 'guest' | 'auth'] =>
      token
        ? ['mainDebatesFeed', user?.id ?? 'auth']
        : ['mainDebatesFeed', 'guest'],
    [token, user?.id],
  );

  const query = useQuery({
    queryKey: mainDebatesFeedQueryKey,
    enabled: isReady,
    queryFn: async (): Promise<UnifiedFeed> => {
      if (token) {
        const data = await fetchDebatesFeed(token, {
          new_limit: 30,
          voted_limit: 30,
        });
        return {kind: 'auth', ...data};
      }
      const pub = await fetchDebatesPublicFeed(30);
      return {kind: 'public', debates: pub.debates};
    },
  });

  const {data, isLoading, isError, error, refetch} = query;

  const onPullRefresh = useCallback(async () => {
    setIsPullRefreshing(true);
    try {
      await refetch();
    } finally {
      setIsPullRefreshing(false);
    }
  }, [refetch]);

  const [feedNowMs, setFeedNowMs] = useState(() => Date.now());
  useEffect(() => {
    const t = setInterval(() => setFeedNowMs(Date.now()), 60_000);
    return () => clearInterval(t);
  }, []);

  const filteredFeed = useMemo((): UnifiedFeed | null => {
    if (!data) {
      return null;
    }
    if (data.kind === 'public') {
      return {
        kind: 'public',
        debates: data.debates.filter(s =>
          debateGeneratedWithinPastSixDays(s, feedNowMs),
        ),
      };
    }
    return {
      kind: 'auth',
      new_debates: data.new_debates.filter(s =>
        debateGeneratedWithinPastSixDays(s, feedNowMs),
      ),
      voted_debates: data.voted_debates.filter(s =>
        debateGeneratedWithinPastSixDays(s, feedNowMs),
      ),
    };
  }, [data, feedNowMs]);

  const {hero, votedList, isGuest} = useMemo(() => {
    if (!filteredFeed) {
      return {
        hero: null as DebateSummary | null,
        votedList: [] as DebateSummary[],
        isGuest: true,
      };
    }
    if (filteredFeed.kind === 'public') {
      const list = filteredFeed.debates;
      return {
        hero: list[0] ?? null,
        votedList: [],
        isGuest: true,
      };
    }
    const n = filteredFeed.new_debates;
    return {
      hero: n[0] ?? null,
      votedList: filteredFeed.voted_debates,
      isGuest: false,
    };
  }, [filteredFeed]);

  /** Warm cache for upcoming heroes (not shown in UI) so the next card does not flash loading. */
  useEffect(() => {
    if (!filteredFeed) return;
    const ids =
      filteredFeed.kind === 'auth'
        ? filteredFeed.new_debates.slice(1, 6).map(s => s.id)
        : filteredFeed.debates.slice(1, 6).map(s => s.id);
    for (const id of ids) {
      if (id > 0) {
        void queryClient.prefetchQuery({
          queryKey: ['debateHero', id],
          queryFn: () => fetchDebateById(id),
        });
      }
    }
  }, [filteredFeed, queryClient]);

  const listSections = useMemo((): MainDebatesSection[] => {
    const newData: FeedRow[] =
      hero != null ? [{kind: 'hero', summary: hero}] : [{kind: 'heroEmpty'}];
    const activityData: FeedRow[] = isGuest
      ? [{kind: 'guestCta'}]
      : votedList.length === 0
        ? [{kind: 'activityEmpty'}]
        : votedList.map(s => ({kind: 'activity' as const, summary: s}));
    return [
      {key: 'new', data: newData},
      {key: 'activity', data: activityData},
    ];
  }, [hero, isGuest, votedList]);

  /** T018/T019: open `SingleDebate` from hero and MY ACTIVITY (fetch full debate by id). */
  const onOpenSummary = useCallback(
    async (summary: DebateSummary) => {
      const full = await fetchDebateById(summary.id);
      if (!full) return;
      const debate: DebateResponse = {
        ...full,
        headline: full.headline ?? summary.headline,
        description: full.description ?? summary.description ?? '',
        cards: full.cards ?? [],
      };
      navigation.navigate('SingleDebate', {
        match: buildPlaceholderMatch(summary),
        debate,
      });
    },
    [navigation],
  );

  /** Immediate hero advance + consensus bump; next card shows without waiting for POST. */
  const onHeroVoteOptimistic = useCallback(
    (detail: DebateHeroVoteSuccessDetail) => {
      queryClient.setQueryData<UnifiedFeed>(
        mainDebatesFeedQueryKey,
        old => {
          if (!old || old.kind !== 'auth') {
            return old;
          }
          const nextNew = old.new_debates.filter(
            s => s.id !== detail.debateId,
          );
          const moved = old.new_debates.find(s => s.id === detail.debateId);
          const movedWithConsensus = moved
            ? withHeroVoteInBinaryConsensus(moved, detail.stance)
            : undefined;
          const nextVoted =
            movedWithConsensus &&
            !old.voted_debates.some(s => s.id === detail.debateId)
              ? [movedWithConsensus, ...old.voted_debates]
              : old.voted_debates;
          return {
            ...old,
            new_debates: nextNew,
            voted_debates: nextVoted,
          };
        },
      );
      const after = queryClient.getQueryData<UnifiedFeed>(
        mainDebatesFeedQueryKey,
      );
      const nextHeroId =
        after?.kind === 'auth' ? after.new_debates[0]?.id : undefined;
      if (nextHeroId != null && nextHeroId !== detail.debateId) {
        void queryClient.prefetchQuery({
          queryKey: ['debateHero', nextHeroId],
          queryFn: () => fetchDebateById(nextHeroId),
        });
      }
    },
    [queryClient, mainDebatesFeedQueryKey],
  );

  const onHeroVoteConfirmed = useCallback(() => {
    void queryClient.invalidateQueries({queryKey: mainDebatesFeedQueryKey});
  }, [queryClient, mainDebatesFeedQueryKey]);

  const onHeroVoteFailed = useCallback(
    (failed: DebateHeroVoteFailedDetail) => {
      queryClient.setQueryData<UnifiedFeed>(
        mainDebatesFeedQueryKey,
        old => {
          if (!old || old.kind !== 'auth') {
            return old;
          }
          const voted = old.voted_debates.filter(
            s => s.id !== failed.debateId,
          );
          const alreadyNew = old.new_debates.some(
            s => s.id === failed.debateId,
          );
          const newDebates = alreadyNew
            ? old.new_debates
            : [failed.summary, ...old.new_debates];
          return {
            ...old,
            new_debates: newDebates,
            voted_debates: voted,
          };
        },
      );
      setVoteErrorToast('Vote failed. Please try again.');
    },
    [queryClient, mainDebatesFeedQueryKey],
  );

  const newSectionEmpty = hero == null;

  const keyExtractor = useCallback((item: FeedRow) => {
    switch (item.kind) {
      case 'hero':
        return `hero-${item.summary.id}`;
      case 'heroEmpty':
        return 'hero-empty';
      case 'guestCta':
        return 'guest-cta';
      case 'activityEmpty':
        return 'activity-empty';
      case 'activity':
        return `activity-${item.summary.id}`;
    }
  }, []);

  const renderSectionHeader = useCallback(
    ({section}: {section: MainDebatesSection}) => {
      if (section.key === 'new') {
        return (
          <View
            style={styles.sectionTitleRow}
            accessibilityRole="header"
            accessibilityLabel="New debates">
            <Text style={styles.sectionTitleLime}>NEW DEBATES</Text>
            <View style={styles.livePill}>
              <View style={styles.liveDot} />
              <Text style={styles.liveText}>LIVE NOW</Text>
            </View>
          </View>
        );
      }
      return (
        <View
          style={[styles.sectionTitleRow, styles.sectionSpacer]}
          accessibilityRole="header"
          accessibilityLabel="My activity">
          <Text style={styles.sectionTitleLime}>MY ACTIVITY</Text>
        </View>
      );
    },
    [],
  );

  const renderItem = useCallback<
    SectionListRenderItem<FeedRow, MainDebatesSection>
  >(
    ({item}) => {
      switch (item.kind) {
        case 'hero':
          return (
            <DebateHeroSwipeCard
              summary={item.summary}
              isLoggedIn={isLoggedIn}
              token={token ?? null}
              onOpen={() => onOpenSummary(item.summary)}
              onVoteOptimistic={onHeroVoteOptimistic}
              onVoteConfirmed={onHeroVoteConfirmed}
              onVoteFailed={onHeroVoteFailed}
              buildPlaceholderMatch={buildPlaceholderMatch}
            />
          );
        case 'heroEmpty':
          return (
            <View
              style={styles.heroEmpty}
              accessibilityLabel="No featured debate in new debates">
              <Text style={styles.heroEmptyTitle}>Nothing new yet</Text>
              <Text style={styles.heroEmptySub}>
                {newSectionEmpty && !isGuest && votedList.length > 0
                  ? 'Check My Activity below — or pull to refresh for new debates.'
                  : 'Pull down to refresh for the latest debates.'}
              </Text>
            </View>
          );
        case 'guestCta':
          return (
            <View
              style={styles.guestActivity}
              accessibilityLabel="Sign in to see your debate activity">
              <Text style={styles.guestCopy}>
                Sign in to see debates you have voted on and track your
                activity.
              </Text>
              <TouchableOpacity
                style={styles.ctaButton}
                onPress={() => rootNavigateToProfileAuth()}
                accessibilityRole="button"
                accessibilityLabel="Sign in">
                <Text style={styles.ctaButtonText}>Sign in</Text>
              </TouchableOpacity>
            </View>
          );
        case 'activityEmpty':
          return (
            <View style={styles.emptyInline}>
              <Text style={styles.muted}>
                No completed debates yet. Swipe-vote on a debate above to see it
                here.
              </Text>
            </View>
          );
        case 'activity':
          return (
            <ActivityDebateCard
              summary={item.summary}
              onPress={() => onOpenSummary(item.summary)}
            />
          );
      }
    },
    [
      isLoggedIn,
      token,
      onOpenSummary,
      onHeroVoteOptimistic,
      onHeroVoteConfirmed,
      onHeroVoteFailed,
      newSectionEmpty,
      isGuest,
      votedList,
    ],
  );

  if (!isReady || (isLoading && !data)) {
    return (
      <View style={[styles.centered, {backgroundColor: BG}]}>
        <StatusBar barStyle="light-content" />
        <ActivityIndicator size="large" color={LIME} />
      </View>
    );
  }

  if (isError) {
    return (
      <View style={[styles.centered, styles.errorWrap, {backgroundColor: BG}]}>
        <StatusBar barStyle="light-content" />
        <Text style={styles.errorText}>{userFacingApiMessage(error)}</Text>
        <TouchableOpacity style={styles.ctaButton} onPress={() => refetch()}>
          <Text style={styles.ctaButtonText}>Try again</Text>
        </TouchableOpacity>
      </View>
    );
  }

  return (
    <View style={styles.root}>
      <StatusBar barStyle="light-content" backgroundColor={BG} />
      {voteErrorToast ? (
        <View
          style={[
            styles.voteToastWrap,
            {paddingBottom: Math.max(insets.bottom, 12) + 8},
          ]}
          pointerEvents="box-none"
          accessibilityLiveRegion="polite"
          accessibilityRole="alert">
          <View style={styles.voteToastInner}>
            <Text style={styles.voteToastText}>{voteErrorToast}</Text>
          </View>
        </View>
      ) : null}
      <SectionList<FeedRow, MainDebatesSection>
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        sections={listSections}
        keyExtractor={keyExtractor}
        renderItem={renderItem}
        renderSectionHeader={renderSectionHeader}
        ListFooterComponent={<View style={styles.listFooterSpacer} />}
        stickySectionHeadersEnabled={false}
        accessibilityLabel="Debates feed"
        refreshControl={
          <RefreshControl
            refreshing={isPullRefreshing}
            onRefresh={onPullRefresh}
            tintColor={LIME}
            progressBackgroundColor={CARD}
          />
        }
        showsVerticalScrollIndicator={false}
        keyboardShouldPersistTaps="handled"
      />
    </View>
  );
};

function ActivityDebateCard({
  summary,
  onPress,
}: {
  summary: DebateSummary;
  onPress: () => void;
}) {
  const handleOpenSource = useCallback(() => {
    const url = summary.source_url?.trim();
    if (!url) return;
    try {
      const parsed = new URL(url);
      const isHttp =
        parsed.protocol === 'http:' || parsed.protocol === 'https:';
      if (!isHttp) return;
      Linking.canOpenURL(url)
        .then(supported => {
          if (supported) {
            Linking.openURL(url).catch(() => {});
          }
        })
        .catch(() => {});
    } catch {
      // Invalid URL, ignore tap.
    }
  }, [summary.source_url]);

  const split = consensusPercents(summary);
  const hasSource =
    !!summary.source_headline?.trim() || !!summary.source_url?.trim();
  const sourceLabel = summary.source_headline?.trim()
    ? summary.source_headline
    : (summary.source_url?.trim() ?? '');
  const a11yConsensus =
    split != null
      ? `Consensus ${split.agreePct}% agree, ${split.disagreePct}% disagree. `
      : '';

  return (
    <TouchableOpacity
      style={styles.activityCard}
      onPress={onPress}
      activeOpacity={0.88}
      accessibilityRole="button"
      accessibilityLabel={`${summary.headline}, voted. ${a11yConsensus}`}
      accessibilityHint="Opens this debate">
      <View style={styles.activityTop}>
        <Text style={styles.activityTitle} numberOfLines={3}>
          {summary.headline}
        </Text>
        <View style={styles.votedPill}>
          <Ionicons name="checkmark-circle" size={14} color={LIME} />
          <Text style={styles.votedPillText}>VOTED</Text>
        </View>
      </View>
      {hasSource ? (
        <Text
          style={styles.activitySource}
          numberOfLines={2}
          onPress={event => {
            event.stopPropagation();
            handleOpenSource();
          }}>
          {sourceLabel}
        </Text>
      ) : null}
      <Text style={styles.consensusLabel}>CONSENSUS</Text>
      <View style={styles.barTrack}>
        {split != null ? (
          <View style={styles.barSplit}>
            <View style={[styles.barSegAgree, {width: `${split.agreePct}%`}]} />
            <View
              style={[styles.barSegDisagree, {width: `${split.disagreePct}%`}]}
            />
          </View>
        ) : null}
      </View>
      <View style={styles.consensusPctRow}>
        {split == null ? (
          <Text style={styles.consensusPctMuted}>—</Text>
        ) : (
          <>
            <Text style={styles.consensusPctAgree}>
              {split.agreePct}% AGREE
            </Text>
            <Text style={styles.consensusPctDisagree}>
              {split.disagreePct}% DISAGREE
            </Text>
          </>
        )}
      </View>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: BG,
  },
  scroll: {
    flex: 1,
  },
  scrollContent: {
    paddingBottom: 8,
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  errorWrap: {
    paddingHorizontal: 24,
  },
  errorText: {
    fontSize: 16,
    color: MUTED,
    textAlign: 'center',
    marginBottom: 16,
  },
  sectionTitleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    marginTop: 4,
    marginBottom: 12,
  },
  sectionSpacer: {
    marginTop: 28,
  },
  sectionTitleLime: {
    fontSize: 13,
    fontWeight: '800',
    letterSpacing: 1.2,
    color: LIME,
  },
  livePill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  liveDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: LIVE_DOT,
  },
  liveText: {
    fontSize: 11,
    fontWeight: '700',
    color: MUTED,
    letterSpacing: 0.8,
  },
  heroEmpty: {
    marginHorizontal: 16,
    padding: 24,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: 'rgba(198,255,0,0.25)',
    borderStyle: 'dashed',
    marginBottom: 12,
  },
  heroEmptyTitle: {
    fontSize: 17,
    fontWeight: '800',
    color: TEXT,
    marginBottom: 8,
  },
  heroEmptySub: {
    fontSize: 14,
    color: MUTED,
    lineHeight: 20,
  },
  activityCard: {
    marginHorizontal: 16,
    marginBottom: 12,
    padding: 16,
    backgroundColor: CARD,
    borderRadius: 14,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
  },
  activityTop: {
    flexDirection: 'row',
    gap: 10,
    marginBottom: 10,
  },
  activityTitle: {
    flex: 1,
    fontSize: 15,
    fontWeight: '700',
    color: TEXT,
    lineHeight: 20,
  },
  votedPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    alignSelf: 'flex-start',
    backgroundColor: 'rgba(198,255,0,0.12)',
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 6,
  },
  votedPillText: {
    fontSize: 10,
    fontWeight: '900',
    color: LIME,
    letterSpacing: 0.5,
  },
  activitySource: {
    fontSize: 12,
    color: LIME,
    opacity: 0.85,
    marginBottom: 10,
  },
  consensusLabel: {
    fontSize: 10,
    fontWeight: '800',
    color: MUTED,
    letterSpacing: 1,
    marginBottom: 6,
  },
  barTrack: {
    height: 8,
    borderRadius: 4,
    backgroundColor: 'rgba(255,255,255,0.08)',
    overflow: 'hidden',
  },
  barSplit: {
    flexDirection: 'row',
    height: '100%',
    width: '100%',
  },
  barSegAgree: {
    height: '100%',
    backgroundColor: LIME,
  },
  barSegDisagree: {
    height: '100%',
    backgroundColor: RED,
  },
  consensusPctRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: 8,
    gap: 8,
  },
  consensusPctAgree: {
    fontSize: 13,
    fontWeight: '800',
    color: LIME,
  },
  consensusPctDisagree: {
    fontSize: 13,
    fontWeight: '800',
    color: RED,
  },
  consensusPctMuted: {
    fontSize: 13,
    fontWeight: '700',
    color: MUTED,
  },
  guestActivity: {
    marginHorizontal: 16,
    padding: 20,
    borderRadius: 14,
    backgroundColor: CARD,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
  },
  guestCopy: {
    fontSize: 14,
    color: MUTED,
    lineHeight: 21,
    marginBottom: 4,
  },
  emptyInline: {
    marginHorizontal: 16,
    padding: 16,
  },
  muted: {
    fontSize: 14,
    color: MUTED,
    lineHeight: 20,
  },
  listFooterSpacer: {
    height: 32,
  },
  voteToastWrap: {
    position: 'absolute',
    left: 16,
    right: 16,
    bottom: 0,
    alignItems: 'center',
    zIndex: 50,
  },
  voteToastInner: {
    backgroundColor: CARD,
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: 'rgba(255,59,48,0.45)',
    maxWidth: '100%',
  },
  voteToastText: {
    color: TEXT,
    fontSize: 14,
    fontWeight: '600',
    textAlign: 'center',
  },
  ctaButton: {
    marginTop: 16,
    alignSelf: 'flex-start',
    backgroundColor: LIME,
    paddingHorizontal: 22,
    paddingVertical: 12,
    borderRadius: 10,
  },
  ctaButtonText: {
    color: BG,
    fontSize: 15,
    fontWeight: '800',
  },
});

export default MainDebatesScreen;
