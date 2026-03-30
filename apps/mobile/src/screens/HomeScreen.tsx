import React, {useMemo, useState} from 'react';
import {
  useWindowDimensions,
  View,
  TouchableOpacity,
  StyleSheet,
  Text,
} from 'react-native';
import {createMaterialTopTabNavigator} from '@react-navigation/material-top-tabs';
import type {NavigationState} from '@react-navigation/native';
import {useNavigation, useRoute} from '@react-navigation/native';
import {LinearGradient} from 'expo-linear-gradient';
import DateScreen from './DateScreen';
import {fetchMatches} from '../services/api';
import {DEFAULT_LEAGUE, type League} from '../constants/leagues';
import {MATCHES_BG, MATCHES_LIME, MATCHES_MUTED} from '../constants/matchesUi';
import {LeagueHorizontalStrip} from '../components/LeagueHorizontalStrip';

type RootTabParamList = {
  [key: string]: undefined;
};

const Tab = createMaterialTopTabNavigator<RootTabParamList>();
const TabNavigator = Tab.Navigator as any;
const TabScreen = Tab.Screen as any;

/** Short labels for league strip (design: PREMIER, etc.) */
function leagueStripLabel(name: string): string {
  const map: Record<string, string> = {
    'Premier League': 'PREMIER',
    'La Liga': 'LA LIGA',
    'Serie A': 'SERIE A',
    Bundesliga: 'BUNDES',
    'Ligue 1': 'LIGUE 1',
    'UEFA Champions League': 'UCL',
    'International Competitions': 'INTL',
  };
  return map[name] ?? name.split(' ')[0]?.toUpperCase() ?? name;
}

const getTabLabel = (date: Date): string => {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const compareDate = new Date(date);
  compareDate.setHours(0, 0, 0, 0);

  if (compareDate.getTime() === today.getTime()) {
    return 'TODAY';
  }

  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);
  if (compareDate.getTime() === yesterday.getTime()) {
    return 'YESTERDAY';
  }

  const tomorrow = new Date(today);
  tomorrow.setDate(today.getDate() + 1);
  if (compareDate.getTime() === tomorrow.getTime()) {
    return 'TOMORROW';
  }

  const days = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT'];
  return `${days[date.getDay()]} ${date.getDate()}`;
};

type DateTabScreenProps = {
  date: Date;
  searchQuery: string;
  selectedLeague?: League | null;
};

const DateTabScreen: React.FC<DateTabScreenProps> = ({
  date,
  searchQuery,
  selectedLeague,
}) => {
  const route = useRoute();
  const navigation = require('@react-navigation/native').useNavigation();
  const currentRoute = (navigation.getState() as NavigationState).routes[
    (navigation.getState() as NavigationState).index
  ].name;
  const isSelected = route.name === currentRoute;
  const [matches, setMatches] = React.useState<any[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);
  const hasLoadedRef = React.useRef(false);
  const isLoadingRef = React.useRef(false);

  const cacheKey = React.useMemo(
    () => `${date.toISOString()}-${selectedLeague?.id || 'all'}`,
    [date, selectedLeague?.id],
  );
  const loadedCacheKeysRef = React.useRef<Set<string>>(new Set());

  React.useEffect(() => {
    if (!loadedCacheKeysRef.current.has(cacheKey)) {
      hasLoadedRef.current = false;
      setMatches([]);
    }

    if (
      isSelected &&
      selectedLeague &&
      !loadedCacheKeysRef.current.has(cacheKey) &&
      !isLoadingRef.current
    ) {
      isLoadingRef.current = true;
      setIsLoading(true);

      const currentCacheKey = cacheKey;
      const currentLeague = selectedLeague;

      const timeoutId = setTimeout(() => {
        if (
          currentCacheKey === cacheKey &&
          isSelected &&
          !loadedCacheKeysRef.current.has(currentCacheKey)
        ) {
          fetchMatches(date, currentLeague.id)
            .then(data => {
              if (currentCacheKey === cacheKey && data) {
                setMatches(data);
                loadedCacheKeysRef.current.add(currentCacheKey);
                hasLoadedRef.current = true;
              }
            })
            .catch(error => {
              console.error('Error loading matches:', error);
            })
            .finally(() => {
              if (currentCacheKey === cacheKey) {
                isLoadingRef.current = false;
                setIsLoading(false);
              }
            });
        } else {
          isLoadingRef.current = false;
          setIsLoading(false);
        }
      }, 100);

      return () => {
        clearTimeout(timeoutId);
      };
    }
  }, [isSelected, date, selectedLeague, cacheKey]);

  const filteredMatches = React.useMemo(() => {
    if (!searchQuery) return matches;

    const query = searchQuery.toLowerCase();
    return matches.filter(
      match =>
        match.teams.home.name.toLowerCase().includes(query) ||
        match.teams.away.name.toLowerCase().includes(query) ||
        match.league.name.toLowerCase().includes(query),
    );
  }, [matches, searchQuery]);

  return (
    <DateScreen
      date={date}
      isSelected={isSelected}
      matches={filteredMatches}
      isLoading={isLoading}
    />
  );
};

const HomeScreen = () => {
  const {width} = useWindowDimensions();
  const [selectedLeague, setSelectedLeague] = useState<League | null>(
    DEFAULT_LEAGUE,
  );

  const dates = useMemo(() => {
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    return [-1, 0, 1].map(offset => {
      const d = new Date(today);
      d.setDate(today.getDate() + offset);
      return d;
    });
  }, []);

  const todayIndex = useMemo(() => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    return dates.findIndex(date => date.getTime() === today.getTime());
  }, [dates]);

  const initialRoute = useMemo(() => {
    const initialDate = dates[todayIndex !== -1 ? todayIndex : 1];
    return `date-${initialDate.toISOString()}`;
  }, [dates, todayIndex]);

  return (
    <View style={styles.root}>
      <TabNavigator
        initialRouteName={initialRoute}
        screenOptions={{
          tabBarScrollEnabled: false,
          tabBarItemStyle: {
            width: width / 3,
          },
          tabBarStyle: {
            backgroundColor: MATCHES_BG,
            elevation: 0,
            shadowOpacity: 0,
            borderBottomWidth: 1,
            borderBottomColor: 'rgba(255,255,255,0.08)',
          },
          tabBarIndicatorStyle: {
            height: 0,
          },
          tabBarActiveTintColor: MATCHES_LIME,
          tabBarInactiveTintColor: MATCHES_MUTED,
          tabBarPressColor: 'rgba(198,255,0,0.12)',
          tabBarPressOpacity: 0.9,
          lazy: true,
        }}>
        {dates.map(date => {
          const dateString = date.toISOString();
          const screenKey = `date-${dateString}`;
          const label = getTabLabel(date);

          return (
            <TabScreen
              key={screenKey}
              name={screenKey}
              options={{
                title: label,
                tabBarLabel: ({focused}: {focused: boolean}) => (
                  <View
                    style={styles.dateTabInner}
                    accessibilityLabel={label}>
                    {focused ? (
                      <LinearGradient
                        colors={['#C6FF00', '#E8FF66']}
                        start={{x: 0, y: 0.5}}
                        end={{x: 1, y: 0.5}}
                        style={StyleSheet.absoluteFill}
                      />
                    ) : null}
                    <Text
                      style={[
                        styles.dateTabText,
                        focused && styles.dateTabTextActive,
                      ]}>
                      {label}
                    </Text>
                  </View>
                ),
                tabBarAccessibilityLabel: `Switch to ${label}`,
              }}>
              {() => (
                <View style={styles.tabBody}>
                  <LeagueHorizontalStrip
                    selectedLeague={selectedLeague}
                    onSelect={setSelectedLeague}
                    includeAllOption={false}
                    accentColor={MATCHES_LIME}
                    mutedColor={MATCHES_MUTED}
                  />
                  <View style={styles.dateContent}>
                    <DateTabScreen
                      date={date}
                      searchQuery=""
                      selectedLeague={selectedLeague}
                    />
                  </View>
                </View>
              )}
            </TabScreen>
          );
        })}
      </TabNavigator>
    </View>
  );
};

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: MATCHES_BG,
  },
  dateTabInner: {
    paddingVertical: 10,
    paddingHorizontal: 4,
    borderRadius: 6,
    overflow: 'hidden',
    minWidth: '100%',
    alignItems: 'center',
    justifyContent: 'center',
  },
  dateTabText: {
    fontSize: 11,
    fontWeight: '800',
    letterSpacing: 0.9,
    color: MATCHES_MUTED,
    zIndex: 1,
  },
  dateTabTextActive: {
    color: '#0B0E14',
  },
  tabBody: {
    flex: 1,
    backgroundColor: MATCHES_BG,
  },
  dateContent: {
    flex: 1,
  },
});

export default HomeScreen;
