import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  ActivityIndicator,
  RefreshControl,
  Image,
} from 'react-native';
import {useNavigation} from '@react-navigation/native';
import {Ionicons} from '@expo/vector-icons';
import type {Match} from '../types/match';
import type {NavigationProp} from '../types/navigation';
import {useMatchNews} from '../hooks/useMatchNews';
import type {NewsArticle} from '../types/news';

interface MatchNewsScreenProps {
  match: Match;
}

// Completed match statuses (FT, AET, PEN, etc.)
const COMPLETED_STATUSES = ['FT', 'AET', 'PEN', 'AWD', 'WO', 'CANC', 'ABD', 'PST'];

function isMatchCompleted(statusShort: string): boolean {
  return COMPLETED_STATUSES.includes(statusShort);
}

// Estimate match end time: kickoff + 105 min (90 min + 15 min half-time)
function getMatchEndTimeISO(fixtureDate: string): string {
  const kickoff = new Date(fixtureDate);
  kickoff.setMinutes(kickoff.getMinutes() + 105);
  return kickoff.toISOString();
}

const MatchNewsScreen: React.FC<MatchNewsScreenProps> = ({match}) => {
  const navigation = useNavigation<NavigationProp>();
  const homeTeam = match.teams.home.name;
  const awayTeam = match.teams.away.name;
  const matchId = match.fixture.id.toString();
  const matchStatus = match.fixture.status.short;
  const matchEndTime = isMatchCompleted(matchStatus)
    ? getMatchEndTimeISO(match.fixture.date)
    : '';

  const {articles, loading, error, refresh, refreshing, invalidateCache} =
    useMatchNews(homeTeam, awayTeam, matchId, matchStatus, matchEndTime);

  const handleRefresh = () => {
    // Invalidate cache to mark as stale and trigger refetch
    // This respects React Query's caching strategy while ensuring fresh data
    invalidateCache();
  };

  const handleNewsItemPress = (url: string) => {
    navigation.navigate('NewsWebView', {url});
  };

  const renderImage = (article: NewsArticle) => {
    if (article.imageUrl) {
      return (
        <Image
          source={{uri: article.imageUrl}}
          style={styles.newsImage}
          resizeMode="cover"
        />
      );
    }

    // Placeholder icon
    return (
      <View style={[styles.newsImage, styles.placeholderImage]}>
        <Ionicons name="football-outline" size={32} color="#999" />
      </View>
    );
  };

  const hasArticles = articles.length > 0;

  if (loading && !hasArticles) {
    return (
      <View style={styles.centerContainer}>
        <ActivityIndicator size="large" color="#007AFF" />
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
          <Ionicons name="refresh" size={20} color="#fff" style={styles.icon} />
          <Text style={styles.retryText}>Retry</Text>
        </TouchableOpacity>
      </View>
    );
  }

  if (!hasArticles) {
    return (
      <View style={styles.centerContainer}>
        <Ionicons name="newspaper-outline" size={48} color="#999" />
        <Text style={styles.noDataText}>
          No news available for {homeTeam} vs {awayTeam}
        </Text>
        <TouchableOpacity style={styles.refreshButton} onPress={handleRefresh}>
          <Ionicons
            name="refresh"
            size={20}
            color="#007AFF"
            style={styles.icon}
          />
          <Text style={styles.refreshText}>Refresh</Text>
        </TouchableOpacity>
      </View>
    );
  }

  return (
    <ScrollView
      style={styles.container}
      refreshControl={
        <RefreshControl
          refreshing={refreshing}
          onRefresh={handleRefresh}
          tintColor="#007AFF"
        />
      }>
      <View style={styles.newsContainer}>
        {/* Match News Section */}
        <View style={[styles.sectionHeader, styles.firstSectionHeader]}>
          <Text style={styles.sectionTitle}>
            {homeTeam} vs {awayTeam}
          </Text>
        </View>
        {articles.map((item: NewsArticle) => (
          <TouchableOpacity
            key={item.id}
            style={styles.newsItem}
            onPress={() => handleNewsItemPress(item.sourceUrl)}
            activeOpacity={0.7}>
            <View style={styles.newsContent}>
              <Text style={styles.newsTitle} numberOfLines={3}>
                {item.title}
              </Text>
              <View style={styles.meta}>
                <Text style={styles.newsPublisher}>{item.sourceName}</Text>
                <Text style={styles.separator}>•</Text>
                <Text style={styles.time}>{item.relativeTime}</Text>
              </View>
            </View>
            {renderImage(item)}
          </TouchableOpacity>
        ))}
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f5f5f5',
  },
  centerContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#f5f5f5',
  },
  loadingText: {
    marginTop: 12,
    fontSize: 16,
    color: '#666',
  },
  errorText: {
    fontSize: 16,
    color: '#ff3b30',
    textAlign: 'center',
    paddingHorizontal: 20,
  },
  noDataText: {
    fontSize: 16,
    color: '#666',
    textAlign: 'center',
    paddingHorizontal: 20,
    marginTop: 16,
    marginBottom: 24,
  },
  retryButton: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#007AFF',
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 8,
    marginTop: 16,
  },
  refreshButton: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#f0f0f0',
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#007AFF',
    marginTop: 16,
  },
  icon: {
    marginRight: 8,
  },
  retryText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  refreshText: {
    color: '#007AFF',
    fontSize: 16,
    fontWeight: '600',
  },
  meta: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: 8,
  },
  separator: {
    fontSize: 12,
    color: '#999',
    marginHorizontal: 6,
  },
  time: {
    fontSize: 12,
    color: '#999',
  },
  newsContainer: {
    padding: 16,
  },
  sectionHeader: {
    marginTop: 8,
    marginBottom: 12,
    paddingBottom: 8,
  },
  firstSectionHeader: {
    marginTop: 0,
  },
  sectionTitle: {
    fontSize: 20,
    fontWeight: '700',
    color: '#333',
    letterSpacing: -0.3,
  },
  newsItem: {
    flexDirection: 'row',
    backgroundColor: '#fff',
    borderRadius: 12,
    marginBottom: 16,
    shadowColor: '#000',
    shadowOffset: {
      width: 0,
      height: 2,
    },
    shadowOpacity: 0.1,
    shadowRadius: 3.84,
    elevation: 5,
    overflow: 'hidden',
    minHeight: 120,
  },
  newsContent: {
    flex: 1,
    padding: 16,
    justifyContent: 'space-between',
  },
  newsTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#333',
    marginBottom: 8,
    lineHeight: 22,
    flexShrink: 1,
  },
  newsPublisher: {
    fontSize: 12,
    color: '#007AFF',
    fontWeight: '500',
  },
  newsImage: {
    width: 120,
    height: '100%',
    minHeight: 120,
    backgroundColor: '#e0e0e0',
  },
  placeholderImage: {
    backgroundColor: '#e0e0e0',
    justifyContent: 'center',
    alignItems: 'center',
  },
});

export default MatchNewsScreen;
