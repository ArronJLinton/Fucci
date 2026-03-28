import React, {useCallback, useMemo} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  RefreshControl,
  ActivityIndicator,
  Linking,
  ScrollView,
  Image,
  StatusBar,
} from 'react-native';
import {LinearGradient} from 'expo-linear-gradient';
import {useQuery} from '@tanstack/react-query';
import {useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
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
import environment from '../config/environment';

/** Design tokens — Velocity Strike–style dark + lime (009 spec references) */
const BG = '#0B0E14';
const LIME = '#C6FF00';
const CARD = '#1A1F2E';
const TEXT = '#FFFFFF';
const MUTED = '#8B92A5';
const RED_X = '#FF3B30';
const LIVE_DOT = '#3B82F6';

/** Stadium / pitch imagery for hero (Unsplash — football) */
const HERO_IMAGE_URI =
  'https://images.unsplash.com/photo-1574629810360-7efbbe195018?w=1200&q=80';

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

/** Visual consensus bar % from feed analytics (proxy — API has no poll % on summaries). */
function consensusPercent(summary: DebateSummary): number {
  const eng = summary.analytics?.engagement_score ?? 0;
  const votes = summary.analytics?.total_votes ?? 0;
  const raw = 42 + eng * 3.2 + Math.min(18, votes * 1.2);
  return Math.min(95, Math.max(22, Math.round(raw)));
}

function debatePillLabel(debateType: string): string {
  return debateType === 'post_match' ? 'CONTROVERSY' : 'PRE-MATCH';
}

const MainDebatesScreen = () => {
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<MainDebatesNavigation>();
  const {isLoggedIn, token, isReady} = useAuth();
  const brandName = environment.APP_NAME.toUpperCase();

  const query = useQuery({
    queryKey: ['mainDebatesFeed', isLoggedIn, token],
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

  const {hero, restNew, votedList, isGuest} = useMemo(() => {
    if (!data) {
      return {
        hero: null as DebateSummary | null,
        restNew: [] as DebateSummary[],
        votedList: [] as DebateSummary[],
        isGuest: true,
      };
    }
    if (data.kind === 'public') {
      const list = data.debates;
      return {
        hero: list[0] ?? null,
        restNew: list.slice(1),
        votedList: [],
        isGuest: true,
      };
    }
    const n = data.new_debates;
    return {
      hero: n[0] ?? null,
      restNew: n.slice(1),
      votedList: data.voted_debates,
      isGuest: false,
    };
  }, [data]);

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

  const goProfile = useCallback(() => {
    navigation.getParent()?.navigate('Profile' as never);
  }, [navigation]);

  if (!isReady || (isLoading && !data)) {
    return (
      <View style={[styles.centered, {paddingTop: insets.top, backgroundColor: BG}]}>
        <StatusBar barStyle="light-content" />
        <ActivityIndicator size="large" color={LIME} />
      </View>
    );
  }

  if (isError) {
    return (
      <View
        style={[
          styles.centered,
          styles.errorWrap,
          {paddingTop: insets.top, backgroundColor: BG},
        ]}>
        <StatusBar barStyle="light-content" />
        <Text style={styles.errorText}>{userFacingApiMessage(error)}</Text>
        <TouchableOpacity style={styles.ctaButton} onPress={() => refetch()}>
          <Text style={styles.ctaButtonText}>Try again</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const showHero = hero != null;
  const newSectionEmpty = !showHero && restNew.length === 0;

  return (
    <View style={[styles.root, {paddingTop: insets.top}]}>
      <StatusBar barStyle="light-content" backgroundColor={BG} />
      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={() => refetch()}
            tintColor={LIME}
            progressBackgroundColor={CARD}
          />
        }
        showsVerticalScrollIndicator={false}>
        {/* Top bar */}
        <View style={styles.topBar}>
          <TouchableOpacity
            onPress={goProfile}
            style={styles.avatarBtn}
            accessibilityRole="button"
            accessibilityLabel="Open profile">
            <Ionicons name="person-circle" size={40} color={LIME} />
          </TouchableOpacity>
          <Text style={styles.brandMark} numberOfLines={1}>
            {brandName}
          </Text>
          <TouchableOpacity
            style={styles.iconBtn}
            accessibilityRole="button"
            accessibilityLabel="Notifications">
            <Ionicons name="notifications-outline" size={26} color={LIME} />
          </TouchableOpacity>
        </View>

        {/* NEW DEBATES */}
        <View style={styles.sectionTitleRow}>
          <Text style={styles.sectionTitleLime}>NEW DEBATES</Text>
          <View style={styles.livePill}>
            <View style={styles.liveDot} />
            <Text style={styles.liveText}>LIVE NOW</Text>
          </View>
        </View>

        {showHero ? (
          <HeroDebateCard
            summary={hero!}
            onOpen={() => onOpenSummary(hero!)}
            onDisagree={() => onOpenSummary(hero!)}
            onAgree={() => onOpenSummary(hero!)}
          />
        ) : (
          <View style={styles.heroEmpty}>
            <Text style={styles.heroEmptyTitle}>Nothing new yet</Text>
            <Text style={styles.heroEmptySub}>
              {newSectionEmpty && !isGuest && votedList.length > 0
                ? 'Check My Activity below — or pull to refresh for new debates.'
                : 'Pull down to refresh for the latest debates.'}
            </Text>
          </View>
        )}

        {restNew.map(s => (
          <CompactDebateRow
            key={s.id}
            summary={s}
            onPress={() => onOpenSummary(s)}
          />
        ))}

        {/* MY ACTIVITY */}
        <View style={[styles.sectionTitleRow, styles.sectionSpacer]}>
          <Text style={styles.sectionTitleLime}>MY ACTIVITY</Text>
        </View>

        {isGuest ? (
          <View style={styles.guestActivity}>
            <Text style={styles.guestCopy}>
              Sign in to see debates you have voted on and track your activity.
            </Text>
            <TouchableOpacity
              style={styles.ctaButton}
              onPress={() => rootNavigate('Login')}
              accessibilityRole="button"
              accessibilityLabel="Sign in">
              <Text style={styles.ctaButtonText}>Sign in</Text>
            </TouchableOpacity>
          </View>
        ) : votedList.length === 0 ? (
          <View style={styles.emptyInline}>
            <Text style={styles.muted}>
              No completed debates yet. Finish voting on all cards in a debate
              to see it here.
            </Text>
          </View>
        ) : (
          votedList.map(s => (
            <ActivityDebateCard
              key={s.id}
              summary={s}
              onPress={() => onOpenSummary(s)}
            />
          ))
        )}

        <View style={{height: 32}} />
      </ScrollView>
    </View>
  );
};

function HeroDebateCard({
  summary,
  onOpen,
  onDisagree,
  onAgree,
}: {
  summary: DebateSummary;
  onOpen: () => void;
  onDisagree: () => void;
  onAgree: () => void;
}) {
  const votes = summary.analytics?.total_votes ?? 0;
  const headline = summary.headline.toUpperCase();

  return (
    <TouchableOpacity
      style={styles.heroOuter}
      onPress={onOpen}
      activeOpacity={0.92}
      accessibilityRole="button"
      accessibilityLabel={summary.headline}>
      <View style={styles.heroClip}>
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
        <View style={styles.heroInner}>
          <View style={styles.pill}>
            <Text style={styles.pillText}>{debatePillLabel(summary.debate_type)}</Text>
          </View>
          <Text style={styles.heroHeadline}>{headline}</Text>
          <View style={styles.heroStats}>
            <View style={styles.statItem}>
              <Ionicons name="time-outline" size={16} color={MUTED} />
              <Text style={styles.statText}>{relativeTimeLabel(summary.created_at)}</Text>
            </View>
            <View style={styles.statItem}>
              <Ionicons name="people-outline" size={16} color={MUTED} />
              <Text style={styles.statText}>{formatVoteCount(votes)} voted</Text>
            </View>
          </View>
          <View style={styles.swipeRow}>
            <TouchableOpacity
              style={styles.swipeBtnRed}
              onPress={onDisagree}
              accessibilityLabel="Open debate — disagree">
              <Ionicons name="close" size={22} color={TEXT} />
            </TouchableOpacity>
            <View style={styles.swipeHint}>
              <View style={styles.swipeLine} />
              <Text style={styles.swipeHintText}>SWIPE TO VOTE</Text>
            </View>
            <TouchableOpacity
              style={styles.swipeBtnLime}
              onPress={onAgree}
              accessibilityLabel="Open debate — agree">
              <Ionicons name="checkmark" size={22} color="#0B0E14" />
            </TouchableOpacity>
          </View>
          <View style={styles.swipeLabels}>
            <Text style={styles.disagreeLabel}>DISAGREE</Text>
            <Text style={styles.agreeLabel}>AGREE</Text>
          </View>
        </View>
      </View>
    </TouchableOpacity>
  );
}

function CompactDebateRow({
  summary,
  onPress,
}: {
  summary: DebateSummary;
  onPress: () => void;
}) {
  const hasSource =
    !!summary.source_headline?.trim() || !!summary.source_url?.trim();
  const sourceLabel = summary.source_headline?.trim()
    ? summary.source_headline
    : summary.source_url?.trim() ?? '';

  return (
    <TouchableOpacity
      style={styles.compactRow}
      onPress={onPress}
      activeOpacity={0.85}>
      <View style={styles.compactTextWrap}>
        <Text style={styles.compactHeadline} numberOfLines={2}>
          {summary.headline}
        </Text>
        {hasSource ? (
          <Text
            style={styles.compactSource}
            numberOfLines={2}
            onPress={() =>
              summary.source_url
                ? Linking.openURL(summary.source_url).catch(() => {})
                : undefined
            }>
            {sourceLabel}
          </Text>
        ) : null}
      </View>
      <Ionicons name="chevron-forward" size={18} color={MUTED} />
    </TouchableOpacity>
  );
}

function ActivityDebateCard({
  summary,
  onPress,
}: {
  summary: DebateSummary;
  onPress: () => void;
}) {
  const pct = consensusPercent(summary);
  const barW = Math.min(100, pct);
  const barColor = pct >= 50 ? LIME : MUTED;
  const hasSource =
    !!summary.source_headline?.trim() || !!summary.source_url?.trim();
  const sourceLabel = summary.source_headline?.trim()
    ? summary.source_headline
    : summary.source_url?.trim() ?? '';

  return (
    <TouchableOpacity
      style={styles.activityCard}
      onPress={onPress}
      activeOpacity={0.88}>
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
        <View
          style={[styles.barFill, {width: `${barW}%`, backgroundColor: barColor}]}
        />
      </View>
      <Text style={styles.consensusPct}>{pct}% AGREE</Text>
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
  topBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  avatarBtn: {
    padding: 4,
  },
  iconBtn: {
    padding: 8,
  },
  brandMark: {
    flex: 1,
    textAlign: 'center',
    fontSize: 15,
    fontWeight: '800',
    fontStyle: 'italic',
    color: LIME,
    letterSpacing: 1.2,
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
  compactRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginHorizontal: 16,
    marginBottom: 8,
    paddingVertical: 12,
    paddingHorizontal: 14,
    backgroundColor: CARD,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
  },
  compactTextWrap: {
    flex: 1,
    paddingRight: 8,
  },
  compactHeadline: {
    fontSize: 15,
    fontWeight: '700',
    color: TEXT,
  },
  compactSource: {
    marginTop: 6,
    fontSize: 12,
    color: LIME,
    opacity: 0.9,
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
