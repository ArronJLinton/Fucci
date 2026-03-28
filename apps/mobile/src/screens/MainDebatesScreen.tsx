import React, {useCallback, useEffect, useMemo} from 'react';
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
  Platform,
} from 'react-native';
import type {SectionListRenderItem} from 'react-native';
import {useQuery, useQueryClient} from '@tanstack/react-query';
import {useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {ScrollView as GHScrollView} from 'react-native-gesture-handler';
import {Ionicons} from '@expo/vector-icons';
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
import {rootNavigate} from '../navigation/rootNavigation';
import DebateHeroSwipeCard, {
  type DebateHeroPanResult,
  type DebateHeroVoteSuccessDetail,
} from '../components/DebateHeroSwipeCard';

/** Design tokens — Velocity Strike–style dark + lime (009 spec references) */
const BG = '#0B0E14';
const LIME = '#C6FF00';
const CARD = '#1A1F2E';
const TEXT = '#FFFFFF';
const MUTED = '#8B92A5';
const LIVE_DOT = '#3B82F6';

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

/** % agree among binary-card upvotes; null when API omits counts or there are no such votes yet. */
function consensusAgreePercent(summary: DebateSummary): number | null {
  const bc = summary.binary_consensus;
  if (!bc) return null;
  const a = bc.agree_upvotes;
  const d = bc.disagree_upvotes;
  const t = a + d;
  if (t <= 0) return null;
  return Math.round((a / t) * 100);
}

function logMainDebatesHero(message: string, payload?: unknown) {
  if (!__DEV__) return;
  if (payload !== undefined) {
    console.log(`[MainDebates hero] ${message}`, payload);
  } else {
    console.log(`[MainDebates hero] ${message}`);
  }
}

const MainDebatesScreen = () => {
  const navigation = useNavigation<MainDebatesNavigation>();
  const queryClient = useQueryClient();
  const {isLoggedIn, token, isReady} = useAuth();

  const onHeroPanResolved = useCallback((result: DebateHeroPanResult) => {
    logMainDebatesHero('pan resolved', result);
  }, []);

  const query = useQuery({
    queryKey: ['mainDebatesFeed', isLoggedIn],
    enabled: isReady,
    queryFn: async (): Promise<UnifiedFeed> => {
      if (isLoggedIn && token) {
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

  const {data, isLoading, isError, error, refetch, isRefetching} = query;

  const {hero, votedList, isGuest} = useMemo(() => {
    if (!data) {
      return {
        hero: null as DebateSummary | null,
        votedList: [] as DebateSummary[],
        isGuest: true,
      };
    }
    if (data.kind === 'public') {
      const list = data.debates;
      return {
        hero: list[0] ?? null,
        votedList: [],
        isGuest: true,
      };
    }
    const n = data.new_debates;
    return {
      hero: n[0] ?? null,
      votedList: data.voted_debates,
      isGuest: false,
    };
  }, [data]);

  /** Warm cache for upcoming heroes (not shown in UI) so the next card does not flash loading. */
  useEffect(() => {
    if (!data) return;
    const ids =
      data.kind === 'auth'
        ? data.new_debates.slice(1, 6).map(s => s.id)
        : data.debates.slice(1, 6).map(s => s.id);
    for (const id of ids) {
      if (id > 0) {
        void queryClient.prefetchQuery({
          queryKey: ['debateHero', id],
          queryFn: () => fetchDebateById(id),
        });
      }
    }
  }, [data, queryClient]);

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
              onPanResolved={onHeroPanResolved}
              onOpen={() => {
                logMainDebatesHero('onOpen (tap / small movement)', {
                  debateId: item.summary.id,
                });
                onOpenSummary(item.summary);
              }}
              onVoteSuccess={(detail: DebateHeroVoteSuccessDetail) => {
                logMainDebatesHero(
                  'SWIPE_VOTE_SUCCEEDED (persisted on server)',
                  {
                    success: true,
                    debateId: detail.debateId,
                    cardId: detail.cardId,
                    voteType: detail.voteType,
                  },
                );
                queryClient.setQueryData<UnifiedFeed>(
                  ['mainDebatesFeed', isLoggedIn],
                  old => {
                    if (!old || old.kind !== 'auth') return old;
                    const nextNew = old.new_debates.filter(
                      s => s.id !== detail.debateId,
                    );
                    const moved = old.new_debates.find(
                      s => s.id === detail.debateId,
                    );
                    const nextVoted =
                      moved &&
                      !old.voted_debates.some(s => s.id === detail.debateId)
                        ? [moved, ...old.voted_debates]
                        : old.voted_debates;
                    return {
                      ...old,
                      new_debates: nextNew,
                      voted_debates: nextVoted,
                    };
                  },
                );
                const after = queryClient.getQueryData<UnifiedFeed>([
                  'mainDebatesFeed',
                  isLoggedIn,
                ]);
                const nextHeroId =
                  after?.kind === 'auth' ? after.new_debates[0]?.id : undefined;
                if (nextHeroId != null && nextHeroId !== detail.debateId) {
                  void queryClient.prefetchQuery({
                    queryKey: ['debateHero', nextHeroId],
                    queryFn: () => fetchDebateById(nextHeroId),
                  });
                }
              }}
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
                onPress={() => rootNavigate('Login')}
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
      onHeroPanResolved,
      onOpenSummary,
      queryClient,
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
      <View
        style={[styles.centered, styles.errorWrap, {backgroundColor: BG}]}>
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
        renderScrollComponent={props => (
          <GHScrollView
            {...props}
            directionalLockEnabled={Platform.OS === 'ios'}
          />
        )}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={() => refetch()}
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
  const pct = consensusAgreePercent(summary);
  const barW = pct == null ? 0 : Math.min(100, pct);
  const barColor = pct == null ? MUTED : pct >= 50 ? LIME : MUTED;
  const hasSource =
    !!summary.source_headline?.trim() || !!summary.source_url?.trim();
  const sourceLabel = summary.source_headline?.trim()
    ? summary.source_headline
    : (summary.source_url?.trim() ?? '');

  return (
    <TouchableOpacity
      style={styles.activityCard}
      onPress={onPress}
      activeOpacity={0.88}
      accessibilityRole="button"
      accessibilityLabel={`${summary.headline}, voted`}
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
          onPress={() =>
            summary.source_url
              ? Linking.openURL(summary.source_url).catch(() => {})
              : undefined
          }>
          {sourceLabel}
        </Text>
      ) : null}
      <Text style={styles.consensusLabel}>CONSENSUS</Text>
      <View style={styles.barTrack}>
        {pct != null ? (
          <View
            style={[
              styles.barFill,
              {width: `${barW}%`, backgroundColor: barColor},
            ]}
          />
        ) : null}
      </View>
      <Text style={styles.consensusPct}>
        {pct == null ? '—' : `${pct}% AGREE`}
      </Text>
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
  barFill: {
    height: '100%',
    borderRadius: 4,
  },
  consensusPct: {
    marginTop: 8,
    fontSize: 13,
    fontWeight: '800',
    color: TEXT,
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
