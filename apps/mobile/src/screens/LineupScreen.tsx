import React, {useState, useEffect} from 'react';
import {
  View,
  Text,
  StyleSheet,
  useWindowDimensions,
  Image,
  ActivityIndicator,
} from 'react-native';
import Animated from 'react-native-reanimated';
import type {Match} from '../types/match';
import {fetchLineup} from '../services/api';
import {useMatchDetailsScroll} from '../context/MatchDetailsScrollContext';

interface PlayerCardProps {
  player: {
    id: number;
    name: string;
    number: number;
    pos: string;
    grid: string;
    photo: string;
  };
  position: {x: number; y: number};
  isAwayTeam: boolean;
}

interface LineupScreenProps {
  match: Match;
}

interface Player {
  id: number;
  name: string;
  number: number;
  pos: string;
  grid: string;
  photo: string;
}

interface LineupData {
  home: {
    starters: Player[];
    substitutes: Player[];
  };
  away: {
    starters: Player[];
    substitutes: Player[];
  };
}

// Helper function to get last name
const getLastName = (fullName: string): string => {
  const names = fullName.split(' ');
  return names[names.length - 1];
};

const PlayerCard: React.FC<PlayerCardProps> = ({
  player,
  position,
  isAwayTeam,
}) => {
  const [imageError, setImageError] = useState(false);

  return (
    <View
      style={[
        styles.playerCard,
        {
          left: `${position.x}%`,
          top: `${position.y}%`,
        },
      ]}>
      <View
        style={[
          styles.playerPhotoContainer,
          isAwayTeam && styles.awayPlayerPhoto,
        ]}>
        {!imageError &&
        (player.photo ||
          'https://media.api-sports.io/football/players/default.png') ? (
          <Image
            source={{
              uri:
                player.photo ||
                'https://media.api-sports.io/football/players/default.png',
            }}
            style={styles.playerPhoto}
            resizeMode="cover"
            onError={() => setImageError(true)}
          />
        ) : (
          <View style={[styles.playerPhoto, styles.placeholderPhoto]} />
        )}
      </View>
      <Text style={[styles.playerName, isAwayTeam && styles.awayPlayerName]}>
        {getLastName(player.name)}
      </Text>
    </View>
  );
};

const SubstituteCard: React.FC<{player: Player}> = ({player}) => {
  const [imageError, setImageError] = useState(false);

  return (
    <View style={styles.substituteCard}>
      {!imageError &&
      (player.photo ||
        'https://media.api-sports.io/football/players/default.png') ? (
        <Image
          source={{
            uri:
              player.photo ||
              'https://media.api-sports.io/football/players/default.png',
          }}
          style={styles.substitutePhoto}
          resizeMode="cover"
          onError={() => setImageError(true)}
        />
      ) : (
        <View style={[styles.substitutePhoto, styles.placeholderPhoto]} />
      )}
      <Text style={styles.substituteName}>{getLastName(player.name)}</Text>
      <Text style={styles.substitutePosition}>{player.pos}</Text>
    </View>
  );
};

const LineupScreen: React.FC<LineupScreenProps> = ({match}) => {
  const {width} = useWindowDimensions();
  const matchScroll = useMatchDetailsScroll();
  const [lineupData, setLineupData] = useState<LineupData | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadLineup = async () => {
      try {
        setIsLoading(true);
        setError(null);

        // Validate match object
        if (!match?.fixture?.id) {
          throw new Error('Invalid match object: missing fixture ID');
        }

        const data = await fetchLineup(match.fixture.id);

        if (data === null) {
          setLineupData(null);
          return;
        }

        setLineupData(data);
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : 'Failed to load lineup data';
        setError(errorMessage);
        console.error('Error fetching lineup:', err);
      } finally {
        setIsLoading(false);
      }
    };

    loadLineup();
  }, [match]);

  if (isLoading) {
    return (
      <View style={styles.centerContainer}>
        <ActivityIndicator size="large" color="#007AFF" />
        <Text style={styles.loadingText}>Loading lineup...</Text>
      </View>
    );
  }

  if (!lineupData) {
    return (
      <View style={styles.centerContainer}>
        <Text style={styles.noDataText}>No lineup data available</Text>
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

  return (
    <Animated.ScrollView
      style={styles.container}
      onScroll={matchScroll?.scrollHandler}
      scrollEventThrottle={16}>
      <View style={[styles.field, {width: width, height: width * 2.0}]}>
        {/* Field markings */}
        <View style={styles.halfwayLine} />
        <View style={styles.centerCircle} />
        <View style={styles.homeBox} />
        <View style={styles.awayBox} />

        {/* Center line marker for visual debugging */}
        <View style={styles.centerLine} />

        {/* Home team goalkeeper */}
        {lineupData.home.starters
          .filter(player => player.pos === 'G')
          .map(player => (
            <PlayerCard
              key={`home-${player.id}`}
              player={player}
              position={{x: 50, y: 2}}
              isAwayTeam={false}
            />
          ))}

        {/* Home team defenders */}
        {lineupData.home.starters
          .filter(player => player.pos === 'D')
          .map((player, index, array) => {
            const totalPlayers = array.length;
            const safeMargin = 15;
            const usableWidth = 100 - 2 * safeMargin;
            const x =
              totalPlayers === 1
                ? 50
                : safeMargin + (usableWidth / (totalPlayers - 1)) * index;
            return (
              <PlayerCard
                key={`home-${player.id}`}
                player={player}
                position={{x, y: 12}}
                isAwayTeam={false}
              />
            );
          })}

        {/* Home team midfielders */}
        {lineupData.home.starters
          .filter(player => player.pos === 'M')
          .map((player, index, array) => {
            const totalPlayers = array.length;
            const safeMargin = 15;
            const usableWidth = 100 - 2 * safeMargin;
            const x =
              totalPlayers === 1
                ? 50
                : safeMargin + (usableWidth / (totalPlayers - 1)) * index;
            return (
              <PlayerCard
                key={`home-${player.id}`}
                player={player}
                position={{x, y: 25}}
                isAwayTeam={false}
              />
            );
          })}

        {/* Home team forwards */}
        {lineupData.home.starters
          .filter(player => player.pos === 'F')
          .map((player, index, array) => {
            const totalPlayers = array.length;
            const safeMargin = 15;
            const usableWidth = 100 - 2 * safeMargin;
            const x =
              totalPlayers === 1
                ? 50
                : safeMargin + (usableWidth / (totalPlayers - 1)) * index;
            return (
              <PlayerCard
                key={`home-${player.id}`}
                player={player}
                position={{x, y: 37}}
                isAwayTeam={false}
              />
            );
          })}

        {/* Away team goalkeeper */}
        {lineupData.away.starters
          .filter(player => player.pos === 'G')
          .map(player => (
            <PlayerCard
              key={`away-${player.id}`}
              player={player}
              position={{x: 50, y: 90}}
              isAwayTeam={true}
            />
          ))}

        {/* Away team defenders */}
        {lineupData.away.starters
          .filter(player => player.pos === 'D')
          .map((player, index, array) => {
            const totalPlayers = array.length;
            const safeMargin = 15;
            const usableWidth = 100 - 2 * safeMargin;
            const x =
              totalPlayers === 1
                ? 50
                : safeMargin + (usableWidth / (totalPlayers - 1)) * index;
            return (
              <PlayerCard
                key={`away-${player.id}`}
                player={player}
                position={{x, y: 80}}
                isAwayTeam={true}
              />
            );
          })}

        {/* Away team midfielders */}
        {lineupData.away.starters
          .filter(player => player.pos === 'M')
          .map((player, index, array) => {
            const totalPlayers = array.length;
            const safeMargin = 15;
            const usableWidth = 100 - 2 * safeMargin;
            const x =
              totalPlayers === 1
                ? 50
                : safeMargin + (usableWidth / (totalPlayers - 1)) * index;
            return (
              <PlayerCard
                key={`away-${player.id}`}
                player={player}
                position={{x, y: 68}}
                isAwayTeam={true}
              />
            );
          })}

        {/* Away team forwards */}
        {lineupData.away.starters
          .filter(player => player.pos === 'F')
          .map((player, index, array) => {
            const totalPlayers = array.length;
            const safeMargin = 15;
            const usableWidth = 100 - 2 * safeMargin;
            const x =
              totalPlayers === 1
                ? 50
                : safeMargin + (usableWidth / (totalPlayers - 1)) * index;
            return (
              <PlayerCard
                key={`away-${player.id}`}
                player={player}
                position={{x, y: 56}}
                isAwayTeam={true}
              />
            );
          })}
      </View>

      {/* Substitutes sections */}
      <View style={styles.substitutesContainer}>
        <View style={styles.substitutesSection}>
          <Text style={styles.substitutesTitle}>
            {match.teams.home.name} Substitutes
          </Text>
          <View style={styles.substitutesGrid}>
            {lineupData.home.substitutes.map(player => (
              <SubstituteCard key={`home-sub-${player.id}`} player={player} />
            ))}
          </View>
        </View>

        <View style={styles.substitutesSection}>
          <Text style={styles.substitutesTitle}>
            {match.teams.away.name} Substitutes
          </Text>
          <View style={styles.substitutesGrid}>
            {lineupData.away.substitutes.map(player => (
              <SubstituteCard key={`away-sub-${player.id}`} player={player} />
            ))}
          </View>
        </View>
      </View>
    </Animated.ScrollView>
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
  },
  noDataText: {
    fontSize: 16,
    color: '#666',
    textAlign: 'center',
  },
  teamsHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: '#fff',
  },
  teamHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
    justifyContent: 'center',
  },
  teamLogo: {
    width: 32,
    height: 32,
    marginRight: 8,
  },
  teamName: {
    fontSize: 16,
    fontWeight: '600',
    color: '#333',
  },
  field: {
    backgroundColor: '#4CAF50',
    position: 'relative',
    overflow: 'hidden',
  },
  halfwayLine: {
    position: 'absolute',
    top: '50%',
    width: '100%',
    height: 2,
    backgroundColor: 'rgba(255, 255, 255, 0.7)',
  },
  centerCircle: {
    position: 'absolute',
    top: '50%',
    left: '50%',
    width: 60,
    height: 60,
    borderRadius: 30,
    borderWidth: 2,
    borderColor: 'rgba(255, 255, 255, 0.7)',
    transform: [{translateX: -30}, {translateY: -30}],
  },
  homeBox: {
    position: 'absolute',
    top: 0,
    left: '25%',
    width: '50%',
    height: '10%',
    borderWidth: 2,
    borderColor: 'rgba(255, 255, 255, 0.7)',
  },
  awayBox: {
    position: 'absolute',
    bottom: 0,
    left: '25%',
    width: '50%',
    height: '10%',
    borderWidth: 2,
    borderColor: 'rgba(255, 255, 255, 0.7)',
  },
  centerLine: {
    position: 'absolute',
    left: '50%',
    top: 0,
    bottom: 0,
    width: 1,
    backgroundColor: 'rgba(255, 255, 255, 0.5)',
  },
  playerCard: {
    position: 'absolute',
    alignItems: 'center',
    justifyContent: 'center',
    width: 100,
    left: '50%',
    marginLeft: -50,
  },
  playerPhotoContainer: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: '#fff',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 4,
    shadowColor: '#000',
    shadowOffset: {
      width: 0,
      height: 2,
    },
    shadowOpacity: 0.25,
    shadowRadius: 3.84,
    elevation: 5,
    overflow: 'hidden',
  },
  awayPlayerPhoto: {
    backgroundColor: '#ffeb3b',
  },
  playerPhoto: {
    width: '100%',
    height: '100%',
    borderRadius: 20,
  },
  playerName: {
    fontSize: 12,
    color: '#fff',
    textShadowColor: 'rgba(0, 0, 0, 0.75)',
    textShadowOffset: {width: -1, height: 1},
    textShadowRadius: 10,
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    paddingHorizontal: 4,
    paddingVertical: 2,
    borderRadius: 4,
    textAlign: 'center',
  },
  awayPlayerName: {
    backgroundColor: 'rgba(0, 0, 0, 0.6)',
  },
  playerPosition: {
    fontSize: 10,
    color: '#fff',
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    paddingHorizontal: 4,
    paddingVertical: 1,
    borderRadius: 2,
    marginTop: 2,
  },
  awayPlayerPosition: {
    backgroundColor: 'rgba(0, 0, 0, 0.6)',
  },
  substitutesContainer: {
    marginTop: 16,
    marginHorizontal: 16,
    marginBottom: 16,
  },
  substitutesSection: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 16,
  },
  substitutesTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#333',
    marginBottom: 12,
  },
  substitutesGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
  },
  substituteCard: {
    backgroundColor: '#f8f8f8',
    borderRadius: 8,
    padding: 8,
    width: '31%',
    alignItems: 'center',
  },
  substitutePhoto: {
    width: 40,
    height: 40,
    borderRadius: 20,
    marginBottom: 8,
  },
  substituteName: {
    fontSize: 12,
    color: '#333',
    textAlign: 'center',
    marginTop: 4,
    fontWeight: '500',
  },
  substitutePosition: {
    fontSize: 10,
    color: '#666',
    marginTop: 2,
  },
  placeholderPhoto: {
    backgroundColor: '#e0e0e0',
    justifyContent: 'center',
    alignItems: 'center',
  },
});

export default LineupScreen;
