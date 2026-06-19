import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import {
  useWindowDimensions,
  View,
  StyleSheet,
  Text,
} from 'react-native';
import {createMaterialTopTabNavigator} from '@react-navigation/material-top-tabs';
import type {NavigationState} from '@react-navigation/native';
import {useNavigation, useRoute} from '@react-navigation/native';
import {LinearGradient} from 'expo-linear-gradient';
import DateScreen from './DateScreen';
import {fetchMatchesForLocalDate} from '../services/api';
import {
  DEFAULT_LEAGUE,
  WORLD_CUP_LEAGUE,
  seasonParamForMatchSearch,
  type League,
} from '../constants/leagues';
import {MATCHES_BG, MATCHES_LIME, MATCHES_MUTED} from '../constants/matchesUi';
import {LeagueHorizontalStrip} from '../components/LeagueHorizontalStrip';
import {resolveHomeScreenDefaultLeague} from '../services/matchesDefaultLeague';
import {WORLD_CUP_ONLY_MODE} from '../config/featureFlags';

type RootTabParamList = {
  [key: string]: undefined;
};

const Tab = createMaterialTopTabNavigator<RootTabParamList>();
const TabNavigator = Tab.Navigator as any;
const TabScreen = Tab.Screen as any;

const TAB_DAY_OFFSETS = [-2, -1, 0, 1, 2] as const;

const WEEKDAY_LABELS = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT'] as const;

const formatCalendarDay = (date: Date): string =>
  `${WEEKDAY_LABELS[date.getDay()]} ${date.getDate()}`;

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

  return formatCalendarDay(date);
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
  const navigation = useNavigation();
  const currentRoute = (navigation.getState() as NavigationState).routes[
    (navigation.getState() as NavigationState).index
  ].name;
  const isSelected = route.name === currentRoute;
  const [matches, setMatches] = React.useState<any[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);
  const isLoadingRef = React.useRef(false);

  const cacheKey = React.useMemo(
    () => `${date.toISOString()}-${selectedLeague?.id || 'all'}`,
    [date, selectedLeague?.id],
  );
  /** In-memory cache per date tab: (date ISO + league id) → matches. Survives league switching (e.g. EPL → La Liga → EPL). */
  const matchesByKeyRef = React.useRef<Map<string, any[]>>(new Map());

  React.useEffect(() => {
    if (!selectedLeague) {
      return;
    }

    const cached = matchesByKeyRef.current.get(cacheKey);

    /** Inactive tab: keep list in sync with (date + league) so we never flash another league after switching strip on a different day tab. */
    if (!isSelected) {
      if (cached !== undefined) {
        setMatches(cached);
      } else {
        setMatches([]);
      }
      setIsLoading(false);
      isLoadingRef.current = false;
      return;
    }

    if (cached !== undefined) {
      setMatches(cached);
      setIsLoading(false);
      isLoadingRef.current = false;
      return;
    }

    isLoadingRef.current = true;
    setIsLoading(true);
    setMatches([]);

    const currentCacheKey = cacheKey;
    const currentLeague = selectedLeague;

    const timeoutId = setTimeout(() => {
      if (currentCacheKey !== cacheKey || !isSelected) {
        isLoadingRef.current = false;
        setIsLoading(false);
        return;
      }

      const hitWhileWaiting = matchesByKeyRef.current.get(currentCacheKey);
      if (hitWhileWaiting !== undefined) {
        setMatches(hitWhileWaiting);
        isLoadingRef.current = false;
        setIsLoading(false);
        return;
      }

      fetchMatchesForLocalDate(
        date,
        currentLeague.id,
        seasonParamForMatchSearch(currentLeague, date),
      )
        .then(data => {
          if (currentCacheKey === cacheKey && data) {
            matchesByKeyRef.current.set(currentCacheKey, data);
            setMatches(data);
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
    }, 100);

    return () => {
      clearTimeout(timeoutId);
    };
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
  const {width: screenWidth} = useWindowDimensions();
  const tabWidth = screenWidth / TAB_DAY_OFFSETS.length;
  const [selectedLeague, setSelectedLeague] = useState<League | null>(
    WORLD_CUP_ONLY_MODE ? WORLD_CUP_LEAGUE : DEFAULT_LEAGUE,
  );
  /** Set when the user picks a league from the strip; blocks async default from overwriting. */
  const userChangedSelectionRef = useRef(false);

  const handleLeagueSelect = useCallback((league: League | null) => {
    userChangedSelectionRef.current = true;
    setSelectedLeague(league);
  }, []);

  useEffect(() => {
    if (WORLD_CUP_ONLY_MODE) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const league = await resolveHomeScreenDefaultLeague(new Date());
        if (!cancelled && !userChangedSelectionRef.current) {
          setSelectedLeague(league);
        }
      } catch {
        /* keep DEFAULT_LEAGUE */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const dates = useMemo(() => {
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    return TAB_DAY_OFFSETS.map(offset => {
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
    const initialDate = dates[todayIndex !== -1 ? todayIndex : 2];
    return `date-${initialDate.toISOString()}`;
  }, [dates, todayIndex]);

  return (
    <View style={styles.root}>
      <TabNavigator
        initialRouteName={initialRoute}
        screenOptions={{
          tabBarScrollEnabled: false,
          tabBarItemStyle: {
            width: tabWidth,
            paddingHorizontal: 0,
            marginHorizontal: 0,
          },
          tabBarStyle: {
            backgroundColor: MATCHES_BG,
            elevation: 0,
            shadowOpacity: 0,
            borderBottomWidth: 1,
            borderBottomColor: 'rgba(255,255,255,0.08)',
            minHeight: 44,
            width: screenWidth,
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
                      ]}
                      numberOfLines={1}
                      allowFontScaling={false}
                      includeFontPadding={false}>
                      {label}
                    </Text>
                  </View>
                ),
                tabBarAccessibilityLabel: `Switch to ${label}`,
              }}>
              {() => (
                <View style={styles.tabBody}>
                  {WORLD_CUP_ONLY_MODE ? null : (
                    <LeagueHorizontalStrip
                      selectedLeague={selectedLeague}
                      onSelect={handleLeagueSelect}
                      includeAllOption={false}
                      accentColor={MATCHES_LIME}
                      mutedColor={MATCHES_MUTED}
                    />
                  )}
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
    paddingVertical: 7,
    paddingHorizontal: 2,
    borderRadius: 5,
    overflow: 'hidden',
    width: '100%',
    minHeight: 32,
    alignItems: 'center',
    justifyContent: 'center',
  },
  dateTabText: {
    fontSize: 12,
    fontWeight: '800',
    letterSpacing: 0,
    color: MATCHES_MUTED,
    zIndex: 1,
    textAlign: 'center',
    width: '100%',
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
