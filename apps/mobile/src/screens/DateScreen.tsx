import React, {useEffect, useRef, useState, useCallback} from 'react';
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

interface DateScreenProps {
  date: Date;
  isSelected?: boolean;
  matches: Match[];
  isLoading?: boolean;
}

const ITEMS_PER_PAGE = 10;

const getMatchTime = (fixtureDate: string): string => {
  const date = new Date(fixtureDate);
  return date.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
};

const getMatchDate = (fixtureDate: string): string => {
  const date = new Date(fixtureDate);
  const months = [
    'JAN',
    'FEB',
    'MAR',
    'APR',
    'MAY',
    'JUN',
    'JUL',
    'AUG',
    'SEP',
    'OCT',
    'NOV',
    'DEC',
  ];
  return `${date.getDate()} ${months[date.getMonth()]}`;
};

const getScoreDisplay = (goals: Match['goals']) => {
  if (goals.home === null || goals.away === null) {
    return 'vs';
  }
  return `${goals.home} - ${goals.away}`;
};

const MatchCard: React.FC<{match: Match}> = ({match}) => {
  const navigation = useNavigation<NavigationProp>();
  const isMountedRef = useRef(true);

  useEffect(() => {
    // Track if component is mounted
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const handlePress = () => {
    if (isMountedRef.current) {
      navigation.navigate('MatchDetails', {match});
    }
  };

  const matchTime = getMatchTime(match.fixture.date.toString());
  const matchDate = getMatchDate(match.fixture.date.toString());
  const venue = match.fixture.venue?.name || '';
  const venueCity = match.fixture.venue?.city || '';
  const statusShort = match.fixture.status?.short ?? '';
  const hasScore =
    match.goals?.home != null &&
    match.goals?.away != null &&
    statusShort !== 'NS';
  const centerPrimary = hasScore
    ? getScoreDisplay(match.goals)
    : matchTime;
  const centerSecondary = hasScore && statusShort !== 'FT' && statusShort !== 'AET' && statusShort !== 'PEN' && statusShort !== 'FT_PEN' && statusShort !== 'AET_PEN'
    ? 'LIVE'
    : matchDate;

  return (
    <TouchableOpacity style={styles.matchCard} onPress={handlePress}>
      <View style={styles.matchInfo}>
        {/* Home Team */}
        <View style={styles.teamContainer}>
          <Image
            source={{uri: match.teams.home.logo}}
            style={styles.teamLogo}
            resizeMode="contain"
          />
          <Text style={styles.teamName} numberOfLines={1}>
            {match.teams.home.name}
          </Text>
        </View>

        {/* Center: Score (if ongoing/over) or Time and Date */}
        <View style={styles.timeDateContainer}>
          <Text style={[styles.matchTime, hasScore && styles.scoreText]}>{centerPrimary}</Text>
          <Text style={styles.matchDate}>{centerSecondary}</Text>
        </View>

        {/* Away Team */}
        <View style={styles.teamContainer}>
          <Image
            source={{uri: match.teams.away.logo}}
            style={styles.teamLogo}
            resizeMode="contain"
          />
          <Text style={styles.teamName} numberOfLines={1}>
            {match.teams.away.name}
          </Text>
        </View>
      </View>

      {/* Venue Information */}
      {(venue || venueCity) && (
        <Text style={styles.venueText} numberOfLines={1}>
          {[venue, venueCity].filter(Boolean).join(', ')}
        </Text>
      )}
    </TouchableOpacity>
  );
};

const DateScreen: React.FC<DateScreenProps> = ({
  date: _date,
  isSelected: _isSelected = false,
  matches = [],
  isLoading = false,
}) => {
  const [displayedCount, setDisplayedCount] = useState(ITEMS_PER_PAGE);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const previousMatchesRef = useRef<Match[]>([]);

  // Reset displayed count when matches change (new data or filter)
  useEffect(() => {
    // Check if matches array has actually changed (different items or count)
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
      // Simulate loading delay for smooth UX
      setTimeout(() => {
        setDisplayedCount(prev =>
          Math.min(prev + ITEMS_PER_PAGE, matches.length),
        );
        setIsLoadingMore(false);
      }, 300);
    }
  }, [hasMore, isLoadingMore, isLoading, matches.length]);

  const renderItem = useCallback(
    ({item}: {item: Match}) => <MatchCard match={item} />,
    [],
  );

  const renderFooter = () => {
    if (!hasMore || isLoading || matches.length === 0) {
      return null;
    }
    return (
      <View style={styles.footerLoader}>
        {isLoadingMore && <ActivityIndicator size="small" color="#1976d2" />}
      </View>
    );
  };

  const renderEmpty = () => {
    if (isLoading) {
      return (
        <View style={styles.loadingContainer}>
          <ActivityIndicator size="large" color="#1976d2" />
          <Text style={styles.loadingText}>Loading matches...</Text>
        </View>
      );
    }

    return (
      <View style={styles.noMatchesContainer}>
        <Text style={styles.noMatchesText}>No matches found</Text>
        <Text style={styles.noMatchesSubText}>
          Try adjusting your search or check another date
        </Text>
      </View>
    );
  };

  const keyExtractor = useCallback(
    (item: Match, index: number) => `${item.fixture.id}-${index}`,
    [],
  );

  return (
    <FlatList
      data={displayedMatches}
      renderItem={renderItem}
      keyExtractor={keyExtractor}
      ListEmptyComponent={renderEmpty}
      ListFooterComponent={renderFooter}
      onEndReached={loadMore}
      onEndReachedThreshold={0.5}
      contentContainerStyle={styles.container}
      style={styles.listContainer}
      showsVerticalScrollIndicator={true}
    />
  );
};

const styles = StyleSheet.create({
  listContainer: {
    flex: 1,
    backgroundColor: '#f5f5f5',
  },
  container: {
    padding: 16,
    flexGrow: 1,
  },
  card: {
    backgroundColor: '#fff',
    borderRadius: 12,
    shadowColor: '#000',
    shadowOffset: {
      width: 0,
      height: 2,
    },
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
    marginBottom: 16,
  },
  selectedCard: {
    backgroundColor: '#e3f2fd',
    borderColor: '#2196f3',
    borderWidth: 2,
    shadowColor: '#2196f3',
    shadowOpacity: 0.2,
    shadowRadius: 8,
    elevation: 5,
  },
  content: {
    padding: 20,
    alignItems: 'center',
  },
  relativeDay: {
    fontSize: 28,
    fontWeight: '700',
    color: '#000',
    marginBottom: 8,
    textAlign: 'center',
  },
  dateText: {
    fontSize: 18,
    fontWeight: '500',
    color: '#000',
    marginBottom: 12,
    textAlign: 'center',
  },
  selectedText: {
    color: '#1976d2',
  },
  matchCard: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    shadowColor: '#000',
    shadowOffset: {
      width: 0,
      height: 1,
    },
    shadowOpacity: 0.08,
    shadowRadius: 2,
    elevation: 2,
  },
  matchInfo: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  teamContainer: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  teamLogo: {
    width: 48,
    height: 48,
    marginBottom: 8,
  },
  teamName: {
    fontSize: 14,
    fontWeight: '500',
    textAlign: 'center',
    color: '#333',
    maxWidth: 100,
  },
  timeDateContainer: {
    alignItems: 'center',
    justifyContent: 'center',
    marginHorizontal: 16,
  },
  matchTime: {
    fontSize: 16,
    fontWeight: '700',
    color: '#FF0000', // Red color for time
    marginBottom: 4,
  },
  scoreText: {
    fontSize: 18,
  },
  matchDate: {
    fontSize: 12,
    color: '#999', // Grey color for date
    textAlign: 'center',
  },
  venueText: {
    fontSize: 12,
    color: '#666',
    textAlign: 'center',
    marginTop: 4,
  },
  loadingContainer: {
    padding: 32,
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 200,
  },
  loadingText: {
    fontSize: 16,
    color: '#666',
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
    fontSize: 18,
    fontWeight: '600',
    color: '#666',
    textAlign: 'center',
    marginBottom: 8,
  },
  noMatchesSubText: {
    fontSize: 14,
    color: '#999',
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
