import React, {useState, useCallback, useMemo, useRef, useEffect} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  ActivityIndicator,
  RefreshControl,
  Image,
  ImageBackground,
  Dimensions,
} from 'react-native';
import {useNavigation} from '@react-navigation/native';
import {Ionicons} from '@expo/vector-icons';
import {LinearGradient} from 'expo-linear-gradient';
import type {NavigationProp} from '../types/navigation';
import {useNews} from '../hooks/useNews';
import type {NewsArticle} from '../types/news';
import type {League} from '../constants/leagues';
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
import {LeagueHorizontalStrip} from '../components/LeagueHorizontalStrip';
import {
  type NewsCategoryId,
  mergeAndSortArticles,
  filterByCategory,
  filterByLeague,
  articleCategoryLabel,
} from '../utils/newsFilters';

const {width: SCREEN_W} = Dimensions.get('window');
const PAGE_PAD = 16;
const GRID_GAP = 10;
const GRID_COL_W = (SCREEN_W - PAGE_PAD * 2 - GRID_GAP) / 2;

const STORY_RINGS: {
  key: string;
  label: string;
  category: NewsCategoryId;
  name: React.ComponentProps<typeof Ionicons>['name'];
}[] = [
  {key: 'goals', label: 'TOP GOALS', category: 'match', name: 'football'},
  {key: 'rumours', label: 'RUMOURS', category: 'transfers', name: 'people'},
  {key: 'matchday', label: 'MATCH DAY', category: 'match', name: 'flash'},
  {key: 'mystory', label: 'MY STORY', category: 'all', name: 'add'},
];

const NewsScreen: React.FC = () => {
  const navigation = useNavigation<NavigationProp>();
  const scrollRef = useRef<ScrollView>(null);
  const [failedImageIds, setFailedImageIds] = useState<Set<string>>(new Set());
  const [category, setCategory] = useState<NewsCategoryId>('all');
  const [leagueFilter, setLeagueFilter] = useState<League | null>(null);
  const {
    todayArticles,
    historyArticles,
    loading,
    error,
    refreshing,
    invalidateCache,
  } = useNews();

  const handleImageError = useCallback((articleId: string) => {
    setFailedImageIds(prev => new Set(prev).add(articleId));
  }, []);

  const handleRefresh = () => {
    setFailedImageIds(new Set());
    invalidateCache();
  };

  const handleNewsItemPress = useCallback(
    (url: string) => {
      navigation.navigate('NewsWebView', {url});
    },
    [navigation],
  );

  const merged = useMemo(
    () => mergeAndSortArticles(todayArticles, historyArticles),
    [todayArticles, historyArticles],
  );

  const filteredArticles = useMemo(() => {
    let list = filterByCategory(merged, category);
    list = filterByLeague(list, leagueFilter);
    return list;
  }, [merged, category, leagueFilter]);

  // Keep the featured card in view when filters change.
  useEffect(() => {
    scrollRef.current?.scrollTo({y: 0, animated: true});
  }, [category, leagueFilter?.id]);

  const featured = filteredArticles[0];
  const gridArticles = filteredArticles.slice(1);

  const onStoryPress = (s: (typeof STORY_RINGS)[0]) => {
    if (s.key === 'mystory') {
      return;
    }
    setCategory(s.category);
  };

  const renderFeatured = (article: NewsArticle) => {
    const imgOk = Boolean(article.imageUrl && !failedImageIds.has(article.id));
    const snippet =
      article.snippet?.trim() ||
      article.title.slice(0, 120) + (article.title.length > 120 ? '…' : '');

    const overlayInner = (
      <View style={styles.featuredStack}>
        <LinearGradient
          colors={['rgba(13,17,23,0.15)', 'rgba(13,17,23,0.55)', '#0D1117']}
          style={StyleSheet.absoluteFill}
        />
        <View style={styles.featuredContent}>
          <View style={styles.exclusivePill}>
            <Text style={styles.exclusiveText}>EXCLUSIVE</Text>
          </View>
          <Text style={styles.featuredTitle} numberOfLines={4}>
            {article.title.toUpperCase()}
          </Text>
          <Text style={styles.featuredSnippet} numberOfLines={2}>
            {snippet}
          </Text>
          <View style={styles.featuredFooter}>
            <View style={styles.authorDot}>
              <Ionicons name="person" size={14} color={NEWS_BG} />
            </View>
            <Text style={styles.featuredByline} numberOfLines={1}>
              BY {article.sourceName.toUpperCase()} •{' '}
              {article.relativeTime.toUpperCase()}
            </Text>
          </View>
        </View>
      </View>
    );

    return (
      <TouchableOpacity
        style={styles.featuredOuter}
        onPress={() => handleNewsItemPress(article.sourceUrl)}
        activeOpacity={0.92}>
        {imgOk ? (
          <ImageBackground
            source={{uri: article.imageUrl!}}
            style={styles.featuredBg}
            imageStyle={styles.featuredBgImg}
            onError={() => handleImageError(article.id)}>
            {overlayInner}
          </ImageBackground>
        ) : (
          <LinearGradient
            colors={['#1c2433', '#0D1117']}
            style={styles.featuredBg}>
            {overlayInner}
          </LinearGradient>
        )}
      </TouchableOpacity>
    );
  };

  const renderGridCard = (article: NewsArticle) => {
    const imgOk = article.imageUrl && !failedImageIds.has(article.id);
    const tag = articleCategoryLabel(article);
    return (
      <TouchableOpacity
        key={article.id}
        style={styles.gridCard}
        onPress={() => handleNewsItemPress(article.sourceUrl)}
        activeOpacity={0.9}>
        <View style={styles.gridImageWrap}>
          {imgOk ? (
            <Image
              source={{uri: article.imageUrl}}
              style={styles.gridImage}
              resizeMode="cover"
              onError={() => handleImageError(article.id)}
            />
          ) : (
            <View style={[styles.gridImage, styles.gridPlaceholder]}>
              <Ionicons name="image-outline" size={32} color={NEWS_MUTED} />
            </View>
          )}
        </View>
        <Text style={styles.gridTitle} numberOfLines={3}>
          {article.title.toUpperCase()}
        </Text>
        <Text style={styles.gridMeta} numberOfLines={2}>
          {tag} • {article.relativeTime.toUpperCase()}
        </Text>
      </TouchableOpacity>
    );
  };

  if (loading && merged.length === 0) {
    return (
      <View style={styles.centerContainer}>
        <ActivityIndicator size="large" color={NEWS_ACCENT} />
        <Text style={styles.loadingText}>Loading news...</Text>
      </View>
    );
  }

  if (error && merged.length === 0) {
    return (
      <View style={styles.centerContainer}>
        <Ionicons name="alert-circle-outline" size={48} color="#ff6b6b" />
        <Text style={styles.errorText}>
          {error?.message || 'Failed to load news'}
        </Text>
        <TouchableOpacity style={styles.retryButton} onPress={handleRefresh}>
          <Ionicons
            name="refresh"
            size={20}
            color={NEWS_BG}
            style={{marginRight: 8}}
          />
          <Text style={styles.retryText}>Retry</Text>
        </TouchableOpacity>
      </View>
    );
  }

  if (!loading && merged.length === 0) {
    return (
      <View style={styles.centerContainer}>
        <Ionicons name="newspaper-outline" size={48} color={NEWS_MUTED} />
        <Text style={styles.noDataText}>No news available right now</Text>
        <TouchableOpacity style={styles.refreshButton} onPress={handleRefresh}>
          <Ionicons
            name="refresh"
            size={20}
            color={NEWS_ACCENT}
            style={{marginRight: 8}}
          />
          <Text style={styles.refreshText}>Refresh</Text>
        </TouchableOpacity>
      </View>
    );
  }

  return (
    <View style={styles.safe}>
      <ScrollView
        ref={scrollRef}
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={handleRefresh}
            tintColor={NEWS_ACCENT}
          />
        }>
        {/* Story rings */}
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.storyRow}>
          {STORY_RINGS.map(s => {
            return (
              <TouchableOpacity
                key={s.key}
                style={styles.storyItem}
                onPress={() => onStoryPress(s)}
                activeOpacity={0.88}>
                <LinearGradient
                  colors={[NEWS_ACCENT, NEWS_CYAN]}
                  start={{x: 0, y: 0}}
                  end={{x: 1, y: 1}}
                  style={styles.storyGradient}>
                  <View style={styles.storyInner}>
                    {s.key === 'mystory' ? (
                      <Ionicons name="add" size={32} color={NEWS_MUTED} />
                    ) : (
                      <Ionicons name={s.name} size={28} color={NEWS_TEXT} />
                    )}
                  </View>
                </LinearGradient>
                <Text style={styles.storyLabel}>{s.label}</Text>
              </TouchableOpacity>
            );
          })}
        </ScrollView>

        <LeagueHorizontalStrip
          selectedLeague={leagueFilter}
          onSelect={setLeagueFilter}
          includeAllOption
          accentColor={NEWS_ACCENT}
          mutedColor={NEWS_MUTED}
        />

        {filteredArticles.length === 0 ? (
          <View style={styles.emptyFilter}>
            <Text style={styles.emptyFilterText}>
              No articles match these filters
            </Text>
            <Text style={styles.emptyFilterHint}>
              Try another story ring or league
            </Text>
          </View>
        ) : (
          <>
            {featured ? renderFeatured(featured) : null}
            <View style={styles.grid}>
              {gridArticles.map(a => (
                <View key={a.id} style={{width: GRID_COL_W}}>
                  {renderGridCard(a)}
                </View>
              ))}
            </View>
          </>
        )}
      </ScrollView>
    </View>
  );
};

const styles = StyleSheet.create({
  safe: {
    flex: 1,
    backgroundColor: NEWS_BG,
  },
  scroll: {
    flex: 1,
    backgroundColor: NEWS_BG,
  },
  scrollContent: {
    paddingBottom: 32,
    paddingTop: 4,
  },
  storyRow: {
    paddingHorizontal: PAGE_PAD,
    paddingTop: 8,
    paddingBottom: 12,
    flexDirection: 'row',
    alignItems: 'flex-start',
  },
  storyItem: {
    alignItems: 'center',
    marginRight: 14,
    width: 78,
  },
  storyGradient: {
    borderRadius: 18,
    padding: 3,
  },
  storyGradientActive: {
    opacity: 1,
  },
  storyInner: {
    width: 72,
    height: 72,
    borderRadius: 15,
    backgroundColor: NEWS_CARD,
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
  },
  storyLabel: {
    marginTop: 8,
    fontSize: 9,
    fontWeight: '800',
    letterSpacing: 0.4,
    color: NEWS_TEXT,
    textAlign: 'center',
  },
  latestRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: PAGE_PAD,
    marginTop: 8,
    marginBottom: 12,
  },
  latestTitle: {
    fontSize: 13,
    fontWeight: '800',
    fontStyle: 'italic',
    letterSpacing: 0.8,
    color: NEWS_TEXT,
  },
  viewAll: {
    fontSize: 11,
    fontWeight: '800',
    letterSpacing: 0.6,
    color: NEWS_ACCENT,
    textDecorationLine: 'underline',
  },
  featuredOuter: {
    marginHorizontal: PAGE_PAD,
    borderRadius: 20,
    overflow: 'hidden',
    marginBottom: 16,
    borderWidth: 1,
    borderColor: NEWS_CARD_BORDER,
  },
  featuredBg: {
    minHeight: 320,
    justifyContent: 'flex-end',
  },
  featuredBgImg: {
    borderRadius: 20,
  },
  featuredStack: {
    flex: 1,
    minHeight: 320,
    justifyContent: 'flex-end',
  },
  featuredContent: {
    padding: 18,
  },
  exclusivePill: {
    alignSelf: 'flex-start',
    backgroundColor: NEWS_EXCLUSIVE,
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 4,
    marginBottom: 10,
  },
  exclusiveText: {
    fontSize: 10,
    fontWeight: '900',
    letterSpacing: 0.6,
    color: '#0D1117',
  },
  featuredTitle: {
    fontSize: 22,
    fontWeight: '900',
    fontStyle: 'italic',
    letterSpacing: 0.5,
    color: NEWS_TEXT,
    lineHeight: 28,
    marginBottom: 10,
  },
  featuredSnippet: {
    fontSize: 14,
    lineHeight: 20,
    color: NEWS_MUTED,
    marginBottom: 14,
  },
  featuredFooter: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  authorDot: {
    width: 28,
    height: 28,
    borderRadius: 14,
    backgroundColor: NEWS_ACCENT,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 10,
  },
  featuredByline: {
    flex: 1,
    fontSize: 10,
    fontWeight: '700',
    letterSpacing: 0.5,
    color: NEWS_TEXT,
  },
  grid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    paddingHorizontal: PAGE_PAD,
    justifyContent: 'space-between',
  },
  gridCard: {
    width: '100%',
    marginBottom: GRID_GAP + 6,
    backgroundColor: NEWS_CARD,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: NEWS_CARD_BORDER,
    overflow: 'hidden',
    paddingBottom: 10,
  },
  gridImageWrap: {
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    overflow: 'hidden',
  },
  gridImage: {
    width: '100%',
    height: 110,
    backgroundColor: 'rgba(255,255,255,0.04)',
  },
  gridPlaceholder: {
    alignItems: 'center',
    justifyContent: 'center',
  },
  gridTitle: {
    fontSize: 12,
    fontWeight: '800',
    fontStyle: 'italic',
    letterSpacing: 0.3,
    color: NEWS_TEXT,
    marginTop: 10,
    paddingHorizontal: 10,
    lineHeight: 16,
  },
  gridMeta: {
    fontSize: 9,
    fontWeight: '700',
    letterSpacing: 0.4,
    color: NEWS_MUTED,
    marginTop: 6,
    paddingHorizontal: 10,
    textTransform: 'uppercase',
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
    fontSize: 16,
    color: NEWS_MUTED,
  },
  errorText: {
    fontSize: 16,
    color: '#ff6b6b',
    textAlign: 'center',
    marginTop: 12,
  },
  noDataText: {
    fontSize: 16,
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
    borderRadius: 8,
    marginTop: 16,
  },
  refreshButton: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: 'rgba(255,255,255,0.08)',
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: NEWS_ACCENT,
    marginTop: 8,
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
  emptyFilter: {
    padding: 24,
    alignItems: 'center',
  },
  emptyFilterText: {
    color: NEWS_TEXT,
    fontSize: 15,
    fontWeight: '600',
    textAlign: 'center',
  },
  emptyFilterHint: {
    color: NEWS_MUTED,
    fontSize: 13,
    marginTop: 8,
    textAlign: 'center',
  },
});

export default NewsScreen;
