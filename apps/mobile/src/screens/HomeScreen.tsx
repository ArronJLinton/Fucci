import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react';
import {
  View,
  StyleSheet,
  Text,
} from 'react-native';
import {createMaterialTopTabNavigator} from '@react-navigation/material-top-tabs';
import type {NavigationState} from '@react-navigation/native';
import {useNavigation, useRoute} from '@react-navigation/native';
import {useQuery} from '@tanstack/react-query';
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
import {matchesForLocalDateQueryKey} from '../queries/keys';
import type {Match} from '../types/match';
import {getMatchRefetchInterval} from '../utils/matchStatus';
import {
  APP_STORE_SCREENSHOT_MODE,
  fetchScreenshotMatchday,
  SCREENSHOT_MATCHDAY,
} from '../demo/screenshotDemo';

type RootTabParamList = {
  [key: string]: undefined;
};

const Tab = createMaterialTopTabNavigator<RootTabParamList>();
const TabNavigator = Tab.Navigator as any;
const TabScreen = Tab.Screen as any;

const TAB_DAY_OFFSETS = [-2, -1, 0, 1, 2] as const;

/** Shared pill width — sized for the longest date tab label ("YESTERDAY"). */
const DATE_TAB_PILL_WIDTH = 86;
const DATE_TAB_ITEM_WIDTH = DATE_TAB_PILL_WIDTH + 6;

const WEEKDAY_LABELS = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT'] as const;

const formatCalendarDay = (date: Date): string =>
  `${WEEKDAY_LABELS[date.getDay()]} ${date.getDate()}`;

const getTabLabel = (date: Date, focusDay?: Date): string => {
  const today = (() => {
    const d = focusDay ? new Date(focusDay) : new Date();
    d.setHours(0, 0, 0, 0);
    return d;
  })();
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

const MATCHES_STALE_MS = 5 * 60 * 1000;
const MATCHES_TODAY_STALE_MS = 2 * 60 * 1000;
/** Poll today's tab while fixtures are live (between manual pull-to-refresh). */
const MATCHES_LIVE_REFETCH_MS = 75 * 1000;

const isSameLocalDay = (a: Date, b: Date): boolean => {
  const left = new Date(a);
  left.setHours(0, 0, 0, 0);
  const right = new Date(b);
  right.setHours(0, 0, 0, 0);
  return left.getTime() === right.getTime();
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

  const leagueId = selectedLeague?.id ?? 0;
  const isTodayTab = isSameLocalDay(date, new Date());
  const {data: matches = [], isLoading, refetch} = useQuery<Match[]>({
    queryKey: matchesForLocalDateQueryKey(date, leagueId),
    queryFn: async () => {
      if (APP_STORE_SCREENSHOT_MODE) {
        return fetchScreenshotMatchday(date);
      }
      if (!selectedLeague) {
        return [];
      }
      const rows = await fetchMatchesForLocalDate(
        date,
        selectedLeague.id,
        seasonParamForMatchSearch(selectedLeague, date),
      );
      return rows ?? [];
    },
    enabled: (APP_STORE_SCREENSHOT_MODE || Boolean(selectedLeague)) && isSelected,
    staleTime: isTodayTab ? MATCHES_TODAY_STALE_MS : MATCHES_STALE_MS,
    gcTime: 15 * 60 * 1000,
    placeholderData: previous => previous,
    refetchOnWindowFocus: isTodayTab,
    refetchInterval: query =>
      APP_STORE_SCREENSHOT_MODE
        ? false
        : isTodayTab && isSelected
          ? getMatchRefetchInterval(
              query.state.data ?? [],
              MATCHES_LIVE_REFETCH_MS,
            )
          : false,
  });

  const onRefresh = useCallback(async () => {
    await refetch();
  }, [refetch]);

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
      onRefresh={onRefresh}
    />
  );
};

const HomeScreen = () => {
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
    const anchor = APP_STORE_SCREENSHOT_MODE
      ? new Date(
          SCREENSHOT_MATCHDAY.getFullYear(),
          SCREENSHOT_MATCHDAY.getMonth(),
          SCREENSHOT_MATCHDAY.getDate(),
        )
      : (() => {
          const now = new Date();
          return new Date(now.getFullYear(), now.getMonth(), now.getDate());
        })();
    return TAB_DAY_OFFSETS.map(offset => {
      const d = new Date(anchor);
      d.setDate(anchor.getDate() + offset);
      return d;
    });
  }, []);

  const todayIndex = useMemo(() => {
    const focus = APP_STORE_SCREENSHOT_MODE
      ? (() => {
          const d = new Date(SCREENSHOT_MATCHDAY);
          d.setHours(0, 0, 0, 0);
          return d;
        })()
      : (() => {
          const today = new Date();
          today.setHours(0, 0, 0, 0);
          return today;
        })();
    return dates.findIndex(date => date.getTime() === focus.getTime());
  }, [dates]);

  const initialRoute = useMemo(() => {
    const initialDate = dates[todayIndex !== -1 ? todayIndex : 2];
    return `date-${initialDate.toISOString()}`;
  }, [dates, todayIndex]);

  const tabFocusDay = APP_STORE_SCREENSHOT_MODE ? SCREENSHOT_MATCHDAY : undefined;

  return (
    <View style={styles.root}>
      <TabNavigator
        initialRouteName={initialRoute}
        screenOptions={{
          tabBarScrollEnabled: true,
          tabBarItemStyle: {
            width: DATE_TAB_ITEM_WIDTH,
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
          const label = getTabLabel(date, tabFocusDay);

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
                    <View style={styles.dateTabPill}>
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
                        allowFontScaling={false}>
                        {label}
                      </Text>
                    </View>
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
    width: DATE_TAB_ITEM_WIDTH,
    minHeight: 32,
    alignItems: 'center',
    justifyContent: 'center',
  },
  dateTabPill: {
    width: DATE_TAB_PILL_WIDTH,
    minHeight: 32,
    paddingVertical: 7,
    paddingHorizontal: 4,
    borderRadius: 5,
    overflow: 'hidden',
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
