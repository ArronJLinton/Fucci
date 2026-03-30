import React, {useMemo, useState, useCallback} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ActivityIndicator,
  RefreshControl,
  Image,
  ImageBackground,
} from 'react-native';
import Animated from 'react-native-reanimated';
import {useNavigation} from '@react-navigation/native';
import {Ionicons} from '@expo/vector-icons';
import {LinearGradient} from 'expo-linear-gradient';
import type {Match} from '../types/match';
import type {NavigationProp} from '../types/navigation';
import {useMatchNews} from '../hooks/useMatchNews';
import type {NewsArticle} from '../types/news';
import {
  NEWS_BG,
  NEWS_CARD,
  NEWS_CARD_BORDER,
  NEWS_ACCENT,
  NEWS_CYAN,
  NEWS_TEXT,
  NEWS_MUTED,
  NEWS_EXCLUSIVE,
} from '../constants/newsUi';
import {articleCategoryLabel} from '../utils/newsFilters';
import {
  useMatchDetailsScroll,
  type MatchDetailsScrollHandler,
} from '../context/MatchDetailsScrollContext';

interface MatchNewsScreenProps {
  match: Match;
  matchScrollHandler?: MatchDetailsScrollHandler;
}

const PAGE_PAD = 16;
const CARD_RADIUS = 12;

// Completed match statuses (FT, AET, PEN, etc.)
const COMPLETED_STATUSES = [
  'FT',
  'AET',
  'PEN',
  'AWD',
  'WO',
  'CANC',
  'ABD',
  'PST',
];

function isMatchCompleted(statusShort: string): boolean {
  return COMPLETED_STATUSES.includes(statusShort);
}

function getMatchEndTimeISO(fixtureDate: string): string {
  const kickoff = new Date(fixtureDate);
  kickoff.setMinutes(kickoff.getMinutes() + 105);
  return kickoff.toISOString();
}

function articleText(a: NewsArticle): string {
  return `${a.title} ${a.snippet || ''}`.toLowerCase();
}

function isStatsArticle(a: NewsArticle): boolean {
  const t = articleText(a);
  return [
    'stat',
    'xg',
    'xga',
    'opta',
    'analytics',
    'possession',
    'expected goal',
    'pass completion',
    'shot',
    'efficiency',
  ].some(k => t.includes(k));
}

function isInjuryArticle(a: NewsArticle): boolean {
  const t = articleText(a);
  return [
    'injury',
    'injured',
    'sideline',
    'knock',
    'hamstring',
    'surgery',
    'ruled out',
    'medical',
    'brace',
    'stretcher',
  ].some(k => t.includes(k));
}

function isPremiumArticle(a: NewsArticle): boolean {
  const t = articleText(a);
  return [
    'analysis',
    'tactical',
    'paradigm',
    'deep dive',
    'premium',
    'insight',
    'column',
    'velocity',
    'masterclass',
  ].some(k => t.includes(k));
}

/** Uppercase tag for standard cards (mock: ATMOSPHERE, POST-MATCH) */
function matchNewsTag(a: NewsArticle): string {
  const t = articleText(a);
  if (
    t.includes('atmosphere') ||
    t.includes('crowd') ||
    t.includes('fans ')
  ) {
    return 'ATMOSPHERE';
  }
  if (
    t.includes('post-match') ||
    t.includes('post match') ||
    t.includes('after the') ||
    t.includes('reaction')
  ) {
    return 'POST-MATCH';
  }
  return articleCategoryLabel(a);
}

function sortByPublishedDesc(list: NewsArticle[]): NewsArticle[] {
  return [...list].sort(
    (a, b) =>
      new Date(b.publishedAt).getTime() - new Date(a.publishedAt).getTime(),
  );
}

const MatchNewsScreen: React.FC<MatchNewsScreenProps> = ({
  match,
  matchScrollHandler,
}) => {
  const navigation = useNavigation<NavigationProp>();
  const matchScroll = useMatchDetailsScroll();
  const onScroll = matchScrollHandler ?? matchScroll?.scrollHandler;
  const homeTeam = match.teams.home.name;
  const awayTeam = match.teams.away.name;
  const matchId = match.fixture.id.toString();
  const matchStatus = match.fixture.status.short;
  const matchEndTime = isMatchCompleted(matchStatus)
    ? getMatchEndTimeISO(match.fixture.date)
    : '';

  const {articles, loading, error, refreshing, invalidateCache} = useMatchNews(
    homeTeam,
    awayTeam,
    matchId,
    matchStatus,
    matchEndTime,
  );

  const [failedImageIds, setFailedImageIds] = useState<Set<string>>(new Set());
  const handleImageError = useCallback((articleId: string) => {
    setFailedImageIds(prev => new Set(prev).add(articleId));
  }, []);

  const sorted = useMemo(
    () => sortByPublishedDesc(articles),
    [articles],
  );

  const layout = useMemo(() => {
    if (sorted.length === 0) {
      return {
        featured: undefined as NewsArticle | undefined,
        statsArticle: undefined as NewsArticle | undefined,
        injuryArticle: undefined as NewsArticle | undefined,
        latest: [] as NewsArticle[],
      };
    }
    const featured = sorted[0];
    const rest = sorted.slice(1);

    const statsArticle = rest.find(isStatsArticle);
    const injuryCandidate = rest.filter(
      a => a.id !== statsArticle?.id,
    );
    const injuryArticle = injuryCandidate.find(isInjuryArticle);

    const skip = new Set<string>();
    if (statsArticle) skip.add(statsArticle.id);
    if (injuryArticle) skip.add(injuryArticle.id);

    const latest = rest.filter(a => !skip.has(a.id));
    return {featured, statsArticle, injuryArticle, latest};
  }, [sorted]);

  const handleRefresh = () => {
    setFailedImageIds(new Set());
    invalidateCache();
  };

  const handleNewsItemPress = (url: string) => {
    navigation.navigate('NewsWebView', {url});
  };

  const hasArticles = articles.length > 0;

  const renderHero = (article: NewsArticle) => {
    const imgOk = Boolean(article.imageUrl && !failedImageIds.has(article.id));
    const snippet =
      article.snippet?.trim() ||
      article.title.slice(0, 140) + (article.title.length > 140 ? '…' : '');

    const overlay = (
      <View style={styles.heroStack}>
        <LinearGradient
          colors={['rgba(10,14,20,0.2)', 'rgba(10,14,20,0.65)', NEWS_BG]}
          style={StyleSheet.absoluteFill}
        />
        <View style={styles.heroContent}>
          <Text style={styles.heroKicker}>MATCH CENTER</Text>
          <Text style={styles.heroTitle} numberOfLines={4}>
            {article.title.toUpperCase()}
          </Text>
          <Text style={styles.heroSnippet} numberOfLines={2}>
            {snippet}
          </Text>
        </View>
      </View>
    );

    return (
      <TouchableOpacity
        style={styles.heroOuter}
        onPress={() => handleNewsItemPress(article.sourceUrl)}
        activeOpacity={0.92}>
        {imgOk ? (
          <ImageBackground
            source={{uri: article.imageUrl!}}
            style={styles.heroBg}
            imageStyle={styles.heroBgImg}
            onError={() => handleImageError(article.id)}>
            {overlay}
          </ImageBackground>
        ) : (
          <LinearGradient colors={['#1a2332', NEWS_BG]} style={styles.heroBg}>
            {overlay}
          </LinearGradient>
        )}
      </TouchableOpacity>
    );
  };

  const renderStatsCard = (article: NewsArticle) => (
    <TouchableOpacity
      key={`stats-${article.id}`}
      style={styles.statsCard}
      onPress={() => handleNewsItemPress(article.sourceUrl)}
      activeOpacity={0.9}>
      <View style={styles.statsRail} />
      <View style={styles.statsBody}>
        <View style={styles.statsHeaderRow}>
          <Ionicons name="analytics-outline" size={16} color={NEWS_CYAN} />
          <Text style={styles.statsLabel}>STATS ANALYSIS</Text>
        </View>
        <Text style={styles.statsHeadline} numberOfLines={3}>
          {article.title}
        </Text>
        <View style={styles.statsFooter}>
          <Text style={styles.statsFooterLeft}>
            {article.sourceName.toUpperCase().includes('OPTA')
              ? 'OPTA STATS'
              : article.sourceName.toUpperCase()}
          </Text>
          <Text style={styles.statsFooterRight}>
            {article.relativeTime.toUpperCase()}
          </Text>
        </View>
      </View>
    </TouchableOpacity>
  );

  const renderInjuryCard = (article: NewsArticle) => {
    const imgOk = Boolean(article.imageUrl && !failedImageIds.has(article.id));
    return (
      <TouchableOpacity
        key={`injury-${article.id}`}
        style={styles.injuryCard}
        onPress={() => handleNewsItemPress(article.sourceUrl)}
        activeOpacity={0.9}>
        {imgOk ? (
          <Image
            source={{uri: article.imageUrl}}
            style={styles.injuryThumb}
            resizeMode="cover"
            onError={() => handleImageError(article.id)}
          />
        ) : (
          <View style={[styles.injuryThumb, styles.injuryThumbPh]}>
            <Ionicons name="medical-outline" size={28} color={NEWS_MUTED} />
          </View>
        )}
        <View style={styles.injuryTextCol}>
          <Text style={styles.injuryHeadline} numberOfLines={3}>
            {article.title.toUpperCase()}
          </Text>
          <View style={styles.injuryFooter}>
            <Text style={styles.injuryMeta}>MEDICAL UPDATE</Text>
            <Text style={styles.injuryDot}>•</Text>
            <Text style={styles.injuryMeta}>
              {article.relativeTime.toUpperCase()}
            </Text>
          </View>
        </View>
      </TouchableOpacity>
    );
  };

  const renderStandardCard = (article: NewsArticle) => {
    const imgOk = article.imageUrl && !failedImageIds.has(article.id);
    const tag = matchNewsTag(article);
    const snippet =
      article.snippet?.trim() ||
      article.title.slice(0, 120) + (article.title.length > 120 ? '…' : '');

    return (
      <TouchableOpacity
        key={article.id}
        style={styles.standardCard}
        onPress={() => handleNewsItemPress(article.sourceUrl)}
        activeOpacity={0.9}>
        <View style={styles.standardImageWrap}>
          {imgOk ? (
            <Image
              source={{uri: article.imageUrl}}
              style={styles.standardImage}
              resizeMode="cover"
              onError={() => handleImageError(article.id)}
            />
          ) : (
            <View style={[styles.standardImage, styles.standardPh]}>
              <Ionicons name="football-outline" size={36} color={NEWS_MUTED} />
            </View>
          )}
          <LinearGradient
            colors={['transparent', 'rgba(10,14,20,0.85)']}
            style={styles.standardImgGradient}
          />
        </View>
        <View style={styles.standardBody}>
          <Text style={styles.standardTag}>{tag}</Text>
          <Text style={styles.standardMeta}>
            {article.sourceName.toUpperCase()} •{' '}
            {article.relativeTime.toUpperCase()}
          </Text>
          <Text style={styles.standardTitle} numberOfLines={3}>
            {article.title}
          </Text>
          <Text style={styles.standardSnippet} numberOfLines={2}>
            {snippet}
          </Text>
        </View>
      </TouchableOpacity>
    );
  };

  const renderPremiumCard = (article: NewsArticle) => {
    const snippet =
      article.snippet?.trim() ||
      article.title.slice(0, 200) + (article.title.length > 200 ? '…' : '');

    return (
      <TouchableOpacity
        key={`premium-${article.id}`}
        style={styles.premiumCard}
        onPress={() => handleNewsItemPress(article.sourceUrl)}
        activeOpacity={0.9}>
        <View style={styles.premiumRail} />
        <View style={styles.premiumBody}>
          <View style={styles.premiumHeaderRow}>
            <Ionicons name="star" size={14} color={NEWS_EXCLUSIVE} />
            <Text style={styles.premiumLabel}>PREMIUM ANALYSIS</Text>
          </View>
          <Text style={styles.premiumTitle} numberOfLines={4}>
            {article.title}
          </Text>
          <Text style={styles.premiumSnippet} numberOfLines={4}>
            {snippet}
          </Text>
          <View style={styles.premiumFooter}>
            <View style={styles.premiumAuthor}>
              <Ionicons name="person-circle-outline" size={18} color={NEWS_MUTED} />
              <Text style={styles.premiumBy} numberOfLines={1}>
                BY {article.sourceName.toUpperCase()}
              </Text>
            </View>
            <Text style={styles.premiumCta}>READ MORE →</Text>
          </View>
        </View>
      </TouchableOpacity>
    );
  };

  if (loading && !hasArticles) {
    return (
      <View style={styles.centerContainer}>
        <ActivityIndicator size="large" color={NEWS_ACCENT} />
        <Text style={styles.loadingText}>Loading news...</Text>
      </View>
    );
  }

  if (error && !hasArticles) {
    return (
      <View style={styles.centerContainer}>
        <Ionicons name="alert-circle-outline" size={48} color="#ff6b6b" />
        <Text style={styles.errorText}>
          {error?.message || 'Failed to load news'}
        </Text>
        <TouchableOpacity style={styles.retryButton} onPress={handleRefresh}>
          <Ionicons name="refresh" size={20} color={NEWS_BG} style={styles.icon} />
          <Text style={styles.retryText}>Retry</Text>
        </TouchableOpacity>
      </View>
    );
  }

  if (!hasArticles) {
    return (
      <View style={styles.centerContainer}>
        <Ionicons name="newspaper-outline" size={48} color={NEWS_MUTED} />
        <Text style={styles.noDataText}>
          No news available for {homeTeam} vs {awayTeam}
        </Text>
        <TouchableOpacity style={styles.refreshButton} onPress={handleRefresh}>
          <Ionicons
            name="refresh"
            size={20}
            color={NEWS_ACCENT}
            style={styles.icon}
          />
          <Text style={styles.refreshText}>Refresh</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const {featured, statsArticle, injuryArticle, latest} = layout;

  let premiumId: string | undefined;
  for (const a of latest) {
    if (isPremiumArticle(a)) {
      premiumId = a.id;
      break;
    }
  }

  return (
    <Animated.ScrollView
      style={styles.container}
      contentContainerStyle={styles.scrollContent}
      showsVerticalScrollIndicator={false}
      onScroll={onScroll}
      scrollEventThrottle={16}
      nestedScrollEnabled
      refreshControl={
        <RefreshControl
          refreshing={refreshing}
          onRefresh={handleRefresh}
          tintColor={NEWS_ACCENT}
        />
      }>
      {featured ? renderHero(featured) : null}
      {statsArticle ? renderStatsCard(statsArticle) : null}
      {injuryArticle ? renderInjuryCard(injuryArticle) : null}

      {latest.length > 0 ? (
        <View style={styles.latestHeader}>
          <Text style={styles.latestTitle}>LATEST UPDATES</Text>
        </View>
      ) : null}

      {latest.map(article => {
        if (article.id === premiumId) {
          return renderPremiumCard(article);
        }
        return renderStandardCard(article);
      })}

      <View style={styles.scrollBottomSpacer} />
    </Animated.ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: NEWS_BG,
  },
  scrollContent: {
    paddingBottom: 8,
  },
  scrollBottomSpacer: {
    height: 24,
  },
  centerContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: NEWS_BG,
    paddingHorizontal: 24,
  },
  loadingText: {
    marginTop: 12,
    fontSize: 15,
    color: NEWS_MUTED,
  },
  errorText: {
    fontSize: 16,
    color: '#ff6b6b',
    textAlign: 'center',
    marginTop: 12,
  },
  noDataText: {
    fontSize: 15,
    color: NEWS_MUTED,
    textAlign: 'center',
    marginTop: 16,
    marginBottom: 24,
  },
  retryButton: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: NEWS_ACCENT,
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 10,
    marginTop: 16,
  },
  refreshButton: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: NEWS_CARD,
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 10,
    borderWidth: 1,
    borderColor: NEWS_CARD_BORDER,
    marginTop: 8,
  },
  icon: {
    marginRight: 8,
  },
  retryText: {
    color: NEWS_BG,
    fontSize: 16,
    fontWeight: '700',
  },
  refreshText: {
    color: NEWS_ACCENT,
    fontSize: 16,
    fontWeight: '700',
  },
  heroOuter: {
    marginHorizontal: PAGE_PAD,
    marginTop: 12,
    borderRadius: CARD_RADIUS,
    overflow: 'hidden',
    borderWidth: 1,
    borderColor: NEWS_CARD_BORDER,
    marginBottom: 14,
  },
  heroBg: {
    minHeight: 280,
    justifyContent: 'flex-end',
  },
  heroBgImg: {
    borderRadius: CARD_RADIUS,
  },
  heroStack: {
    flex: 1,
    minHeight: 280,
    justifyContent: 'flex-end',
  },
  heroContent: {
    padding: 16,
  },
  heroKicker: {
    fontSize: 10,
    fontWeight: '900',
    letterSpacing: 1.2,
    color: NEWS_ACCENT,
    marginBottom: 8,
  },
  heroTitle: {
    fontSize: 20,
    fontWeight: '900',
    letterSpacing: 0.4,
    color: NEWS_TEXT,
    lineHeight: 26,
    marginBottom: 8,
  },
  heroSnippet: {
    fontSize: 13,
    lineHeight: 18,
    color: NEWS_MUTED,
  },
  statsCard: {
    marginHorizontal: PAGE_PAD,
    flexDirection: 'row',
    backgroundColor: NEWS_CARD,
    borderRadius: CARD_RADIUS,
    borderWidth: 1,
    borderColor: NEWS_CARD_BORDER,
    marginBottom: 12,
    overflow: 'hidden',
  },
  statsRail: {
    width: 4,
    backgroundColor: NEWS_CYAN,
  },
  statsBody: {
    flex: 1,
    padding: 14,
  },
  statsHeaderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    marginBottom: 8,
  },
  statsLabel: {
    fontSize: 11,
    fontWeight: '800',
    letterSpacing: 0.8,
    color: NEWS_CYAN,
  },
  statsHeadline: {
    fontSize: 16,
    fontWeight: '800',
    color: NEWS_TEXT,
    lineHeight: 22,
    marginBottom: 10,
  },
  statsFooter: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  statsFooterLeft: {
    fontSize: 10,
    fontWeight: '700',
    letterSpacing: 0.5,
    color: NEWS_MUTED,
  },
  statsFooterRight: {
    fontSize: 10,
    fontWeight: '700',
    color: NEWS_MUTED,
  },
  injuryCard: {
    marginHorizontal: PAGE_PAD,
    flexDirection: 'row',
    backgroundColor: NEWS_CARD,
    borderRadius: CARD_RADIUS,
    borderWidth: 1,
    borderColor: NEWS_CARD_BORDER,
    marginBottom: 16,
    overflow: 'hidden',
  },
  injuryThumb: {
    width: 96,
    height: 96,
    backgroundColor: '#1c2433',
  },
  injuryThumbPh: {
    justifyContent: 'center',
    alignItems: 'center',
  },
  injuryTextCol: {
    flex: 1,
    padding: 12,
    justifyContent: 'space-between',
  },
  injuryHeadline: {
    fontSize: 13,
    fontWeight: '800',
    letterSpacing: 0.3,
    color: NEWS_TEXT,
    lineHeight: 18,
  },
  injuryFooter: {
    flexDirection: 'row',
    alignItems: 'center',
    flexWrap: 'wrap',
    marginTop: 8,
  },
  injuryMeta: {
    fontSize: 10,
    fontWeight: '700',
    color: NEWS_MUTED,
    letterSpacing: 0.4,
  },
  injuryDot: {
    fontSize: 10,
    color: NEWS_MUTED,
    marginHorizontal: 6,
  },
  latestHeader: {
    paddingHorizontal: PAGE_PAD,
    marginBottom: 12,
    marginTop: 4,
  },
  latestTitle: {
    fontSize: 14,
    fontWeight: '900',
    fontStyle: 'italic',
    letterSpacing: 1,
    color: NEWS_ACCENT,
  },
  standardCard: {
    marginHorizontal: PAGE_PAD,
    backgroundColor: NEWS_CARD,
    borderRadius: CARD_RADIUS,
    borderWidth: 1,
    borderColor: NEWS_CARD_BORDER,
    marginBottom: 14,
    overflow: 'hidden',
  },
  standardImageWrap: {
    position: 'relative',
  },
  standardImage: {
    width: '100%',
    height: 160,
    backgroundColor: '#1c2433',
  },
  standardPh: {
    justifyContent: 'center',
    alignItems: 'center',
  },
  standardImgGradient: {
    ...StyleSheet.absoluteFillObject,
    height: 80,
    top: undefined,
    bottom: 0,
  },
  standardBody: {
    padding: 14,
  },
  standardTag: {
    fontSize: 11,
    fontWeight: '900',
    letterSpacing: 0.6,
    color: NEWS_TEXT,
    marginBottom: 6,
  },
  standardMeta: {
    fontSize: 10,
    fontWeight: '700',
    color: NEWS_MUTED,
    marginBottom: 8,
    letterSpacing: 0.3,
  },
  standardTitle: {
    fontSize: 16,
    fontWeight: '800',
    color: NEWS_TEXT,
    lineHeight: 22,
    marginBottom: 6,
  },
  standardSnippet: {
    fontSize: 13,
    color: NEWS_MUTED,
    lineHeight: 19,
  },
  premiumCard: {
    marginHorizontal: PAGE_PAD,
    flexDirection: 'row',
    backgroundColor: NEWS_CARD,
    borderRadius: CARD_RADIUS,
    borderWidth: 1,
    borderColor: NEWS_CARD_BORDER,
    marginBottom: 14,
    overflow: 'hidden',
  },
  premiumRail: {
    width: 4,
    backgroundColor: NEWS_EXCLUSIVE,
  },
  premiumBody: {
    flex: 1,
    padding: 14,
  },
  premiumHeaderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    marginBottom: 8,
  },
  premiumLabel: {
    fontSize: 11,
    fontWeight: '800',
    letterSpacing: 0.7,
    color: NEWS_EXCLUSIVE,
  },
  premiumTitle: {
    fontSize: 16,
    fontWeight: '800',
    color: NEWS_TEXT,
    lineHeight: 22,
    marginBottom: 8,
  },
  premiumSnippet: {
    fontSize: 13,
    color: NEWS_MUTED,
    lineHeight: 19,
    marginBottom: 12,
  },
  premiumFooter: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 8,
  },
  premiumAuthor: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
    gap: 6,
  },
  premiumBy: {
    fontSize: 11,
    fontWeight: '700',
    color: NEWS_MUTED,
    flexShrink: 1,
  },
  premiumCta: {
    fontSize: 12,
    fontWeight: '800',
    color: NEWS_TEXT,
    letterSpacing: 0.3,
  },
});

export default MatchNewsScreen;
