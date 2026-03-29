import React, {useEffect, useRef, useState, useCallback, useMemo} from 'react';
import {
  Text,
  StyleSheet,
  View,
  FlatList,
  Image,
  ActivityIndicator,
  TouchableOpacity,
} from 'react-native';
import {useNavigation} from '@react-navigation/native';
import type {Match} from '../types/match';
import type {NavigationProp} from '../types/navigation';
import {
  MATCHES_BG,
  MATCHES_LIME,
  MATCHES_ORANGE,
  MATCHES_CARD,
  MATCHES_CARD_BORDER,
  MATCHES_MUTED,
  MATCHES_TEXT,
} from '../constants/matchesUi';

interface DateScreenProps {
  date: Date;
  isSelected?: boolean;
  matches: Match[];
  isLoading?: boolean;
}

const ITEMS_PER_PAGE = 10;

const LIVE_STATUSES = new Set(['1H', '2H', 'HT', 'ET', 'P', 'BT']);
const FINISHED_STATUSES = new Set([
  'FT',
  'AET',
  'PEN',
  'FT_PEN',
  'AET_PEN',
]);

function isLiveStatus(short: string): boolean {
  return LIVE_STATUSES.has(short);
}

function isFinishedStatus(short: string): boolean {
  return FINISHED_STATUSES.has(short);
}

const getMatchTime = (fixtureDate: string): string => {
  const d = new Date(fixtureDate);
  return d.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
};

function minutesUntilKickoff(fixtureDate: string): number {
  const t = new Date(fixtureDate).getTime();
  return Math.ceil((t - Date.now()) / 60000);
}

function formatVenueUpper(match: Match): string {
  const v = match.fixture.venue?.name || '';
  const c = match.fixture.venue?.city || '';
  return [v, c].filter(Boolean).join(', ').toUpperCase();
}

function statusLeftRight(match: Match): {
  left: string;
  tone: 'live' | 'ht' | 'upcoming' | 'done';
} {
  const s = match.fixture.status.short;
  const elapsed = match.fixture.status.elapsed ?? 0;

  if (s === 'HT') {
    return {left: '• HALF TIME', tone: 'ht'};
  }
  if (isLiveStatus(s)) {
    return {left: `• LIVE • ${elapsed}'`, tone: 'live'};
  }
  if (s === 'NS' || s === 'TBD') {
    const mins = minutesUntilKickoff(match.fixture.date);
    if (mins > 0 && mins < 24 * 60) {
      return {left: `STARTS IN ${mins}M`, tone: 'upcoming'};
    }
    return {left: 'UPCOMING', tone: 'upcoming'};
  }
  return {left: s, tone: 'done'};
}

function SectionTitle({isToday}: {isToday: boolean}) {
  return (
    <View style={styles.sectionRow}>
      <View style={styles.sectionBar} />
      <Text style={styles.sectionTitle}>
        {isToday ? "TODAY'S FIXTURES" : 'FIXTURES'}
      </Text>
    </View>
  );
}

const MatchCard: React.FC<{match: Match; featuredLayout: boolean}> = ({
  match,
  featuredLayout,
}) => {
  const navigation = useNavigation<NavigationProp>();
  const isMountedRef = useRef(true);

  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const handlePress = () => {
    if (isMountedRef.current) {
      navigation.navigate('MatchDetails', {match});
    }
  };

  const venue = formatVenueUpper(match);
  const {left, tone} = statusLeftRight(match);
  const s = match.fixture.status.short;
  const h = match.goals.home;
  const a = match.goals.away;
  const hasScore =
    h != null && a != null && (isLiveStatus(s) || isFinishedStatus(s) || s === 'HT');
  const homeLeading = h != null && a != null && h > a;
  const awayLeading = h != null && a != null && a > h;

  if (featuredLayout && (s === 'HT' || tone === 'ht')) {
    return (
      <TouchableOpacity
        style={[styles.card, styles.cardFeatured]}
        onPress={handlePress}
        activeOpacity={0.9}>
        <View style={styles.cardTopRow}>
          <Text style={styles.statusOrange}>{left}</Text>
          <View style={styles.featuredBadge}>
            <Text style={styles.featuredBadgeText}>★ FEATURED</Text>
          </View>
        </View>
        <View style={styles.featuredRows}>
          <View style={styles.featuredTeamRow}>
            <Image
              source={{uri: match.teams.home.logo}}
              style={styles.teamLogoSm}
              resizeMode="contain"
            />
            <Text style={styles.teamNameFeatured} numberOfLines={1}>
              {match.teams.home.name}
            </Text>
            {hasScore && (
              <Text
                style={[
                  styles.scoreFeatured,
                  homeLeading && styles.scoreLeading,
                ]}>
                {h}
              </Text>
            )}
          </View>
          <View style={styles.featuredTeamRow}>
            <Image
              source={{uri: match.teams.away.logo}}
              style={styles.teamLogoSm}
              resizeMode="contain"
            />
            <Text style={styles.teamNameFeatured} numberOfLines={1}>
              {match.teams.away.name}
            </Text>
            {hasScore && (
              <Text
                style={[
                  styles.scoreFeatured,
                  awayLeading && styles.scoreLeading,
                ]}>
                {a}
              </Text>
            )}
          </View>
        </View>
      </TouchableOpacity>
    );
  }

  if (tone === 'upcoming' || (s === 'NS' && !hasScore)) {
    const kick = getMatchTime(match.fixture.date);
    return (
      <TouchableOpacity style={styles.card} onPress={handlePress}>
        <View style={styles.cardTopRow}>
          <Text style={styles.statusOrange}>{left}</Text>
          <Text style={styles.venueTop} numberOfLines={1}>
            {venue}
          </Text>
        </View>
        <View style={styles.matchInfo}>
          <View style={styles.teamContainer}>
            <Image
              source={{uri: match.teams.home.logo}}
              style={styles.teamLogo}
              resizeMode="contain"
            />
            <Text style={styles.teamName} numberOfLines={2}>
              {match.teams.home.name}
            </Text>
          </View>
          <View style={styles.kickoffBox}>
            <Text style={styles.kickoffText}>{kick}</Text>
          </View>
          <View style={styles.teamContainer}>
            <Image
              source={{uri: match.teams.away.logo}}
              style={styles.teamLogo}
              resizeMode="contain"
            />
            <Text style={styles.teamName} numberOfLines={2}>
              {match.teams.away.name}
            </Text>
          </View>
        </View>
      </TouchableOpacity>
    );
  }

  return (
    <TouchableOpacity style={styles.card} onPress={handlePress}>
      <View style={styles.cardTopRow}>
        <Text style={styles.statusOrange}>{left}</Text>
        <Text style={styles.venueTop} numberOfLines={1}>
          {venue}
        </Text>
      </View>
      <View style={styles.matchInfo}>
        <View style={styles.teamContainer}>
          <Image
            source={{uri: match.teams.home.logo}}
            style={styles.teamLogo}
            resizeMode="contain"
          />
          <Text style={styles.teamName} numberOfLines={2}>
            {match.teams.home.name}
          </Text>
        </View>
        <View style={styles.scoreCenter}>
          {hasScore ? (
            <View style={styles.scoreRow}>
              <Text
                style={[
                  styles.scoreNum,
                  homeLeading && styles.scoreLeading,
                  !homeLeading && !awayLeading && styles.scoreTie,
                ]}>
                {h}
              </Text>
              <Text style={styles.scoreSep}> : </Text>
              <Text
                style={[
                  styles.scoreNum,
                  awayLeading && styles.scoreLeading,
                  !homeLeading && !awayLeading && styles.scoreTie,
                ]}>
                {a}
              </Text>
            </View>
          ) : (
            <Text style={styles.vsText}>vs</Text>
          )}
        </View>
        <View style={styles.teamContainer}>
          <Image
            source={{uri: match.teams.away.logo}}
            style={styles.teamLogo}
            resizeMode="contain"
          />
          <Text style={styles.teamName} numberOfLines={2}>
            {match.teams.away.name}
          </Text>
        </View>
      </View>
    </TouchableOpacity>
  );
};

const DateScreen: React.FC<DateScreenProps> = ({
  date,
  isSelected: _isSelected = false,
  matches = [],
  isLoading = false,
}) => {
  const [displayedCount, setDisplayedCount] = useState(ITEMS_PER_PAGE);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const previousMatchesRef = useRef<Match[]>([]);

  const isToday = useMemo(() => {
    const a = new Date(date);
    a.setHours(0, 0, 0, 0);
    const b = new Date();
    b.setHours(0, 0, 0, 0);
    return a.getTime() === b.getTime();
  }, [date]);

  useEffect(() => {
    const matchesChanged =
      previousMatchesRef.current.length !== matches.length ||
      previousMatchesRef.current.some(
        (prev, index) => prev.fixture.id !== matches[index]?.fixture.id,
      );

    if (matchesChanged) {
      setDisplayedCount(ITEMS_PER_PAGE);
      previousMatchesRef.current = matches;
    }
  }, [matches]);

  const displayedMatches = matches.slice(0, displayedCount);
  const hasMore = displayedCount < matches.length;

  const loadMore = useCallback(() => {
    if (hasMore && !isLoadingMore && !isLoading) {
      setIsLoadingMore(true);
      setTimeout(() => {
        setDisplayedCount(prev =>
          Math.min(prev + ITEMS_PER_PAGE, matches.length),
        );
        setIsLoadingMore(false);
      }, 300);
    }
  }, [hasMore, isLoadingMore, isLoading, matches.length]);

  const renderItem = useCallback(({item}: {item: Match}) => {
    return (
      <MatchCard
        match={item}
        featuredLayout={item.fixture.status.short === 'HT'}
      />
    );
  }, []);

  const renderHeader = useCallback(() => {
    if (matches.length === 0) {
      return null;
    }
    return (
      <View style={styles.listHeader}>
        <SectionTitle isToday={isToday} />
      </View>
    );
  }, [matches.length, isToday]);

  const renderFooter = () => {
    if (!hasMore || isLoading || matches.length === 0) {
      return null;
    }
    return (
      <View style={styles.footerLoader}>
        {isLoadingMore && (
          <ActivityIndicator size="small" color={MATCHES_LIME} />
        )}
      </View>
    );
  };

  const renderEmpty = useCallback(() => {
    if (isLoading) {
      return (
        <View style={styles.loadingContainer}>
          <ActivityIndicator size="large" color={MATCHES_LIME} />
          <Text style={styles.loadingText}>Loading matches...</Text>
        </View>
      );
    }
    return (
      <View style={styles.noMatchesContainer}>
        <Text style={styles.noMatchesText}>No matches found</Text>
        <Text style={styles.noMatchesSubText}>
          Try another day or league
        </Text>
      </View>
    );
  }, [isLoading]);

  const keyExtractor = useCallback(
    (item: Match, index: number) => `${item.fixture.id}-${index}`,
    [],
  );

  return (
    <FlatList
      data={displayedMatches}
      renderItem={renderItem}
      keyExtractor={keyExtractor}
      ListEmptyComponent={
        matches.length === 0 ? renderEmpty : undefined
      }
      ListHeaderComponent={renderHeader}
      ListFooterComponent={renderFooter}
      onEndReached={loadMore}
      onEndReachedThreshold={0.5}
      contentContainerStyle={styles.container}
      style={styles.listContainer}
      showsVerticalScrollIndicator={false}
    />
  );
};

const styles = StyleSheet.create({
  listContainer: {
    flex: 1,
    backgroundColor: MATCHES_BG,
  },
  container: {
    paddingHorizontal: 16,
    paddingBottom: 24,
    flexGrow: 1,
  },
  listHeader: {
    marginBottom: 8,
  },
  sectionRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
    marginTop: 4,
  },
  sectionBar: {
    width: 4,
    height: 22,
    backgroundColor: MATCHES_LIME,
    marginRight: 10,
    borderRadius: 2,
  },
  sectionTitle: {
    color: MATCHES_TEXT,
    fontSize: 13,
    fontWeight: '800',
    fontStyle: 'italic',
    letterSpacing: 1,
  },
  card: {
    backgroundColor: MATCHES_CARD,
    borderRadius: 14,
    padding: 14,
    marginBottom: 12,
    borderWidth: 1,
    borderColor: MATCHES_CARD_BORDER,
  },
  cardFeatured: {
    borderLeftWidth: 4,
    borderLeftColor: MATCHES_LIME,
  },
  cardTopRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  statusOrange: {
    color: MATCHES_ORANGE,
    fontSize: 11,
    fontWeight: '800',
    letterSpacing: 0.4,
    flex: 1,
  },
  venueTop: {
    color: MATCHES_MUTED,
    fontSize: 10,
    fontWeight: '700',
    letterSpacing: 0.6,
    maxWidth: '48%',
    textAlign: 'right',
  },
  featuredBadge: {
    backgroundColor: 'rgba(255, 193, 7, 0.2)',
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: 4,
  },
  featuredBadgeText: {
    color: '#FFC107',
    fontSize: 10,
    fontWeight: '800',
  },
  featuredRows: {},
  featuredTeamRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
  },
  teamLogoSm: {
    width: 36,
    height: 36,
    marginRight: 10,
  },
  teamNameFeatured: {
    flex: 1,
    color: MATCHES_TEXT,
    fontSize: 15,
    fontWeight: '700',
  },
  scoreFeatured: {
    color: MATCHES_TEXT,
    fontSize: 22,
    fontWeight: '800',
    minWidth: 28,
    textAlign: 'right',
  },
  matchInfo: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  teamContainer: {
    flex: 1,
    alignItems: 'center',
  },
  teamLogo: {
    width: 48,
    height: 48,
    marginBottom: 8,
  },
  teamName: {
    fontSize: 12,
    fontWeight: '600',
    textAlign: 'center',
    color: MATCHES_TEXT,
    maxWidth: 110,
  },
  scoreCenter: {
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 8,
    minWidth: 72,
  },
  scoreRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  scoreNum: {
    fontSize: 22,
    fontWeight: '800',
    color: MATCHES_TEXT,
  },
  scoreSep: {
    fontSize: 18,
    fontWeight: '600',
    color: MATCHES_MUTED,
  },
  scoreLeading: {
    color: MATCHES_LIME,
  },
  scoreTie: {
    color: MATCHES_TEXT,
  },
  vsText: {
    fontSize: 14,
    fontWeight: '700',
    color: MATCHES_MUTED,
  },
  kickoffBox: {
    backgroundColor: '#0B0E14',
    paddingHorizontal: 14,
    paddingVertical: 10,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: MATCHES_CARD_BORDER,
  },
  kickoffText: {
    color: MATCHES_LIME,
    fontSize: 16,
    fontWeight: '800',
    letterSpacing: 0.5,
  },
  loadingContainer: {
    padding: 32,
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 200,
  },
  loadingText: {
    fontSize: 15,
    color: MATCHES_MUTED,
    marginTop: 16,
    textAlign: 'center',
  },
  noMatchesContainer: {
    padding: 32,
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 200,
  },
  noMatchesText: {
    fontSize: 17,
    fontWeight: '600',
    color: MATCHES_TEXT,
    textAlign: 'center',
    marginBottom: 8,
  },
  noMatchesSubText: {
    fontSize: 14,
    color: MATCHES_MUTED,
    textAlign: 'center',
  },
  footerLoader: {
    paddingVertical: 20,
    alignItems: 'center',
    justifyContent: 'center',
  },
});

export default React.memo(DateScreen, (prev, next) => {
  return (
    prev.date.getTime() === next.date.getTime() &&
    prev.isSelected === next.isSelected &&
    prev.matches === next.matches &&
    prev.isLoading === next.isLoading
  );
});
