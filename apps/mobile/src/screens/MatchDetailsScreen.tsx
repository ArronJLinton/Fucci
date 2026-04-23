import React, {useState, useEffect, useRef, useMemo} from 'react';
import {
  View,
  Text,
  StyleSheet,
  Image,
  ImageBackground,
  TouchableOpacity,
  useWindowDimensions,
} from 'react-native';
import {useRoute, useNavigation, RouteProp} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import {createMaterialTopTabNavigator} from '@react-navigation/material-top-tabs';
import {StatusBar} from 'expo-status-bar';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import {Ionicons} from '@expo/vector-icons';
import Animated, {
  useSharedValue,
  useAnimatedStyle,
  useAnimatedScrollHandler,
  interpolate,
  Extrapolation,
} from 'react-native-reanimated';
import type {RootStackParamList} from '../types/navigation';
import LineupScreen from './LineupScreen';
import MatchNewsScreen from './MatchNewsScreen';
import DebateScreen from './DebateScreen';
import {TableScreen} from './TableScreen';
import {generateDebateSet} from '../services/api';
import {MatchDetailsScrollProvider} from '../context/MatchDetailsScrollContext';
import {
  MATCH_CENTER_BG,
  MATCH_CENTER_BLACK,
  MATCH_CENTER_LIME,
  MATCH_CENTER_CYAN,
  MATCH_CENTER_TEXT,
  MATCH_CENTER_TAB_INACTIVE,
  HERO_EXPANDED_HEIGHT,
  HERO_COLLAPSED_HEIGHT,
  HERO_COLLAPSE_SCROLL_RANGE,
} from '../constants/matchCenterUi';
import type {Match} from '../types/match';
import {snapchatUsernameForTeamName} from '../config/matchSnapchatAccounts';

const MAX_PRELOAD_KEYS = 64;
const preloadFiredFor = new Set<string>();
const preloadFiredOrder: string[] = [];
const preloadInFlight = new Set<string>();

function preloadKey(matchId: number, debateType: string): string {
  return `${matchId}:${debateType}`;
}

function markPreloadFired(key: string) {
  if (preloadFiredFor.has(key)) return;
  if (preloadFiredOrder.length >= MAX_PRELOAD_KEYS) {
    const oldest = preloadFiredOrder.shift();
    if (oldest) preloadFiredFor.delete(oldest);
  }
  preloadFiredOrder.push(key);
  preloadFiredFor.add(key);
}

type MatchDetailsRouteProp = RouteProp<RootStackParamList, 'MatchDetails'>;
const Tab = createMaterialTopTabNavigator();
const TabNavigator = Tab.Navigator as any;
const TabScreen = Tab.Screen as any;

function statusPillLabel(match: Match): string {
  const short = match.fixture.status.short;
  const long = match.fixture.status.long ?? '';
  const elapsed = match.fixture.status.elapsed;

  if (['1H', '2H', 'ET'].includes(short) && elapsed != null) {
    return `LIVE ${elapsed}'`;
  }
  const map: Record<string, string> = {
    NS: 'NOT STARTED',
    TBD: 'NOT STARTED',
    '1H': 'FIRST HALF',
    HT: 'HALF TIME',
    '2H': 'SECOND HALF',
    ET: 'EXTRA TIME',
    BT: 'BREAK TIME',
    PEN: 'PENALTIES',
    FT: 'FULL TIME',
    AET: 'AFTER EXTRA TIME',
    FT_PEN: 'FULL TIME',
    AET_PEN: 'AFTER EXTRA TIME',
    PST: 'POSTPONED',
    CANC: 'CANCELLED',
    ABD: 'ABANDONED',
    AWD: 'AWARDED',
    WO: 'WALKOVER',
    LIVE: 'LIVE',
  };
  return map[short] ?? long.toUpperCase().slice(0, 28);
}

function isLiveStatus(short: string): boolean {
  return ['1H', '2H', 'ET', 'BT', 'PEN', 'LIVE'].includes(short);
}

const MatchDetailsScreen = () => {
  const route = useRoute<MatchDetailsRouteProp>();
  const navigation =
    useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const {width} = useWindowDimensions();
  const insets = useSafeAreaInsets();
  const match = route.params.match;
  const [homeLogoError, setHomeLogoError] = useState(false);
  const [awayLogoError, setAwayLogoError] = useState(false);
  const isMountedRef = useRef(true);

  const scrollY = useSharedValue(0);
  const scrollHandler = useAnimatedScrollHandler({
    onScroll: event => {
      const y = event.contentOffset.y;
      // Avoid negative offsets (rubber-band) fighting hero interpolation
      scrollY.value = y < 0 ? 0 : y;
    },
  });

  const scrollContextValue = useMemo(() => ({scrollHandler}), [scrollHandler]);

  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    const matchId = match?.fixture?.id;
    if (!matchId) return;
    const status = match?.fixture?.status?.short ?? '';
    const finished = ['FT', 'AET', 'PEN', 'FT_PEN', 'AET_PEN'].includes(status);
    const debateType = finished ? 'post_match' : 'pre_match';
    const key = preloadKey(matchId, debateType);
    if (preloadFiredFor.has(key) || preloadInFlight.has(key)) return;
    preloadInFlight.add(key);
    generateDebateSet(matchId, debateType, 3)
      .then(result => {
        if (result != null && !result.rateLimited) {
          markPreloadFired(key);
        }
      })
      .catch(() => {})
      .finally(() => {
        preloadInFlight.delete(key);
      });
  }, [match?.fixture?.id, match?.fixture?.status?.short]);

  const heroAnimatedStyle = useAnimatedStyle(() => {
    const h = interpolate(
      scrollY.value,
      [0, HERO_COLLAPSE_SCROLL_RANGE],
      [HERO_EXPANDED_HEIGHT, HERO_COLLAPSED_HEIGHT],
      Extrapolation.CLAMP,
    );
    return {
      height: h,
      overflow: 'hidden' as const,
    };
  });

  /** Fades and collapses expanded score row so compact strip can dominate when scrolled */
  const expandedScoreStyle = useAnimatedStyle(() => {
    const r = HERO_COLLAPSE_SCROLL_RANGE;
    return {
      opacity: interpolate(
        scrollY.value,
        [0, r * 0.58],
        [1, 0],
        Extrapolation.CLAMP,
      ),
      maxHeight: interpolate(
        scrollY.value,
        [0, r * 0.92],
        [200, 0],
        Extrapolation.CLAMP,
      ),
      overflow: 'hidden' as const,
    };
  });

  const heroCompactOpacityStyle = useAnimatedStyle(() => {
    const r = HERO_COLLAPSE_SCROLL_RANGE;
    return {
      opacity: interpolate(
        scrollY.value,
        [r * 0.33, r * 1.05],
        [0, 1],
        Extrapolation.CLAMP,
      ),
    };
  });

  const statusLabel = statusPillLabel(match);
  const live = isLiveStatus(match.fixture.status.short);

  const homeSnapUser = useMemo(
    () => snapchatUsernameForTeamName(match.teams.home.name),
    [match.teams.home.name],
  );
  const awaySnapUser = useMemo(
    () => snapchatUsernameForTeamName(match.teams.away.name),
    [match.teams.away.name],
  );

  const openTeamSnapchatStories = (side: 'home' | 'away') => {
    const u = side === 'home' ? homeSnapUser : awaySnapUser;
    if (!u) {
      return;
    }
    navigation.navigate('MatchSnapchatStories', {
      snapchatUsername: u,
      teamDisplayName:
        side === 'home' ? match.teams.home.name : match.teams.away.name,
    });
  };

  const MatchHero = () => (
    <Animated.View style={[styles.heroOuter, heroAnimatedStyle]}>
      <ImageBackground
        source={require('../../assets/images/stadium_background.jpeg')}
        style={styles.heroImageBg}
        resizeMode="cover">
        <View style={styles.heroScrim} pointerEvents="none" />
        <View style={styles.heroInner}>
          <View style={styles.heroTopRow}>
            <TouchableOpacity
              onPress={() => navigation.goBack()}
              hitSlop={{top: 12, bottom: 12, left: 12, right: 12}}
              accessibilityRole="button"
              accessibilityLabel="Go back">
              <Ionicons
                name="chevron-back"
                size={26}
                color={MATCH_CENTER_LIME}
              />
            </TouchableOpacity>
            <View pointerEvents="none">
              <Ionicons
                name="ellipsis-vertical"
                size={22}
                color={MATCH_CENTER_LIME}
              />
            </View>
          </View>

          <Animated.View style={[styles.heroScoreBlock, expandedScoreStyle]}>
            <TouchableOpacity
              style={[styles.teamBlock, styles.teamBlockSide]}
              activeOpacity={homeSnapUser ? 0.88 : 1}
              disabled={!homeSnapUser}
              onPress={() => openTeamSnapchatStories('home')}
              hitSlop={homeSnapUser ? {top: 8, bottom: 8, left: 4, right: 4} : undefined}
              accessibilityRole="button"
              accessibilityLabel={
                homeSnapUser
                  ? `Open ${match.teams.home.name} Snapchat story`
                  : `Home team ${match.teams.home.name}`
              }>
              {!homeLogoError && match.teams.home.logo ? (
                <Image
                  source={{uri: match.teams.home.logo}}
                  style={styles.badge}
                  resizeMode="contain"
                  onError={() => {
                    if (isMountedRef.current) setHomeLogoError(true);
                  }}
                />
              ) : (
                <View style={[styles.badge, styles.badgePlaceholder]} />
              )}
              <Text style={styles.teamName} numberOfLines={2}>
                {match.teams.home.name.toUpperCase()}
              </Text>
            </TouchableOpacity>

            <View style={[styles.scoreMid, styles.scoreMidCenter]}>
              <Text style={styles.scoreText}>
                {match.goals.home ?? 0} - {match.goals.away ?? 0}
              </Text>
              <View style={[styles.statusPill, live && styles.statusPillLive]}>
                <Text
                  style={[
                    styles.statusPillText,
                    live && styles.statusPillTextLive,
                  ]}>
                  {statusLabel}
                </Text>
              </View>
            </View>

            <TouchableOpacity
              style={[styles.teamBlock, styles.teamBlockSide]}
              activeOpacity={awaySnapUser ? 0.88 : 1}
              disabled={!awaySnapUser}
              onPress={() => openTeamSnapchatStories('away')}
              hitSlop={awaySnapUser ? {top: 8, bottom: 8, left: 4, right: 4} : undefined}
              accessibilityRole="button"
              accessibilityLabel={
                awaySnapUser
                  ? `Open ${match.teams.away.name} Snapchat story`
                  : `Away team ${match.teams.away.name}`
              }>
              {!awayLogoError && match.teams.away.logo ? (
                <Image
                  source={{uri: match.teams.away.logo}}
                  style={styles.badge}
                  resizeMode="contain"
                  onError={() => {
                    if (isMountedRef.current) setAwayLogoError(true);
                  }}
                />
              ) : (
                <View style={[styles.badge, styles.badgePlaceholder]} />
              )}
              <Text style={styles.teamName} numberOfLines={2}>
                {match.teams.away.name.toUpperCase()}
              </Text>
            </TouchableOpacity>
          </Animated.View>

          <Animated.View
            style={[styles.heroCompactRow, heroCompactOpacityStyle]}
            pointerEvents="none">
            <Text style={styles.heroCompactScore} numberOfLines={1}>
              {match.teams.home.name.slice(0, 3).toUpperCase()}{' '}
              <Text style={styles.heroCompactNums}>
                {match.goals.home ?? 0}-{match.goals.away ?? 0}
              </Text>{' '}
              {match.teams.away.name.slice(0, 3).toUpperCase()}
            </Text>
          </Animated.View>
        </View>
      </ImageBackground>
    </Animated.View>
  );

  return (
    <View
      style={[
        styles.root,
        // Bleed into parent SafeArea top padding so hero/tabs sit higher; insets keep controls clear of notch.
        {marginTop: -insets.top, paddingTop: insets.top},
      ]}>
      <StatusBar style="light" />
      <MatchDetailsScrollProvider value={scrollContextValue}>
        <MatchHero />
        <View style={styles.tabsWrap}>
          <TabNavigator
            initialRouteName="News"
            screenListeners={{
              tabPress: () => {
                scrollY.value = 0;
              },
              focus: () => {
                scrollY.value = 0;
              },
            }}
            style={styles.tabNavigator}
            screenOptions={{
              // Pager horizontal swipe competes with vertical ScrollViews; disable for smoother scroll + hero sync.
              swipeEnabled: false,
              tabBarScrollEnabled: true,
              tabBarItemStyle: {
                width: width / 4,
                alignItems: 'center',
                justifyContent: 'center',
              },
              tabBarStyle: {
                backgroundColor: MATCH_CENTER_BG,
                elevation: 0,
                shadowOpacity: 0,
                borderBottomWidth: StyleSheet.hairlineWidth,
                borderBottomColor: 'rgba(255,255,255,0.08)',
              },
              tabBarIndicatorStyle: {
                backgroundColor: MATCH_CENTER_LIME,
                height: 3,
              },
              tabBarActiveTintColor: MATCH_CENTER_LIME,
              tabBarInactiveTintColor: MATCH_CENTER_TAB_INACTIVE,
              tabBarLabelStyle: {
                fontWeight: '800',
                fontSize: 10,
                letterSpacing: 0.6,
                textTransform: 'uppercase',
              },
              tabBarPressColor: 'rgba(223,255,0,0.12)',
              tabBarPressOpacity: 0.85,
            }}>
            <TabScreen
              name="Lineup"
              options={{
                tabBarLabel: 'Lineup',
              }}>
              {() => (
                <LineupScreen
                  match={match}
                  matchScrollHandler={scrollHandler}
                />
              )}
            </TabScreen>
            <TabScreen
              name="Table"
              options={{
                tabBarLabel: 'Table',
              }}>
              {() => (
                <TableScreen match={match} matchScrollHandler={scrollHandler} />
              )}
            </TabScreen>
            <TabScreen
              name="News"
              options={{
                tabBarLabel: 'News',
              }}>
              {() => (
                <MatchNewsScreen
                  match={match}
                  matchScrollHandler={scrollHandler}
                />
              )}
            </TabScreen>
            <TabScreen
              name="Debate"
              options={{
                tabBarLabel: 'Debate',
              }}>
              {() => (
                <DebateScreen
                  match={match}
                  stackNavigation={navigation}
                  matchScrollHandler={scrollHandler}
                />
              )}
            </TabScreen>
          </TabNavigator>
        </View>
      </MatchDetailsScrollProvider>
    </View>
  );
};

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: MATCH_CENTER_BG,
  },
  /** Fills space under hero so tab scenes get a bounded height and vertical ScrollViews actually scroll. */
  tabsWrap: {
    flex: 1,
    minHeight: 0,
  },
  tabNavigator: {
    flex: 1,
  },
  heroOuter: {
    backgroundColor: MATCH_CENTER_BLACK,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: 'rgba(255,255,255,0.08)',
  },
  heroImageBg: {
    flex: 1,
    width: '100%',
  },
  heroScrim: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(6, 8, 12, 0.62)',
  },
  heroInner: {
    flex: 1,
    paddingHorizontal: 16,
    paddingBottom: 6,
    zIndex: 1,
  },
  heroTopRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 4,
    minHeight: 28,
    paddingTop: 0,
  },
  heroScoreBlock: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    minHeight: 0,
  },
  teamBlock: {
    alignItems: 'center',
  },
  teamBlockSide: {
    flex: 1,
    justifyContent: 'center',
  },
  scoreMidCenter: {
    flex: 1,
    justifyContent: 'center',
  },
  badge: {
    width: 44,
    height: 44,
    borderRadius: 8,
    marginBottom: 4,
    backgroundColor: 'rgba(255,255,255,0.04)',
  },
  badgePlaceholder: {
    backgroundColor: 'rgba(255,255,255,0.08)',
  },
  teamName: {
    fontSize: 10,
    fontWeight: '800',
    color: MATCH_CENTER_TEXT,
    textAlign: 'center',
    letterSpacing: 0.3,
  },
  scoreMid: {
    alignItems: 'center',
    paddingHorizontal: 6,
  },
  scoreText: {
    fontSize: 24,
    fontWeight: '800',
    color: MATCH_CENTER_TEXT,
    letterSpacing: 1,
    textShadowColor: 'rgba(0,0,0,0.85)',
    textShadowOffset: {width: 0, height: 1},
    textShadowRadius: 10,
  },
  statusPill: {
    marginTop: 4,
    paddingHorizontal: 12,
    paddingVertical: 5,
    borderRadius: 999,
    borderWidth: 1,
    borderColor: MATCH_CENTER_CYAN,
    backgroundColor: 'rgba(0,229,255,0.08)',
  },
  statusPillLive: {
    borderColor: MATCH_CENTER_LIME,
    backgroundColor: 'rgba(223,255,0,0.1)',
  },
  statusPillText: {
    fontSize: 10,
    fontWeight: '800',
    letterSpacing: 0.8,
    color: MATCH_CENTER_CYAN,
  },
  statusPillTextLive: {
    color: MATCH_CENTER_LIME,
  },
  heroCompactRow: {
    position: 'absolute',
    left: 16,
    right: 16,
    bottom: 8,
    alignItems: 'center',
  },
  heroCompactScore: {
    fontSize: 13,
    fontWeight: '800',
    color: MATCH_CENTER_TEXT,
    letterSpacing: 0.5,
  },
  heroCompactNums: {
    color: MATCH_CENTER_LIME,
    fontSize: 15,
  },
});

export default MatchDetailsScreen;
