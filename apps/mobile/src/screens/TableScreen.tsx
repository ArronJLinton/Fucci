import React, {useEffect, useMemo, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ActivityIndicator,
  Image,
  Pressable,
} from 'react-native';
import Animated from 'react-native-reanimated';
import {Ionicons} from '@expo/vector-icons';
import type {Match} from '../types/match';
import {fetchStandings} from '../services/api';
import {
  useMatchDetailsScroll,
  type MatchDetailsScrollHandler,
} from '../context/MatchDetailsScrollContext';
import {
  MATCH_CENTER_BG,
  MATCH_CENTER_CARD,
  MATCH_CENTER_LIME,
  MATCH_CENTER_CYAN,
  MATCH_CENTER_MUTED,
  MATCH_CENTER_TEXT,
} from '../constants/matchCenterUi';

const PAGE = 16;
const PREVIEW_ROWS = 6;

type Standing = {
  rank: number;
  team: {
    id: number;
    name: string;
    logo: string;
  };
  points: number;
  goalsDiff?: number;
  group?: string;
  all: {
    played: number;
    win: number;
    draw: number;
    lose: number;
    goals: {
      for: number;
      against: number;
    };
  };
  goalDifference?: number;
};

function normName(s: string): string {
  return s.trim().toLowerCase();
}

function goalDiff(s: Standing): number {
  if (s.goalsDiff != null) return s.goalsDiff;
  if (s.goalDifference != null) return s.goalDifference;
  return s.all.goals.for - s.all.goals.against;
}

function formatGD(v: number): string {
  if (v > 0) return `+${v}`;
  return String(v);
}

function formatSeasonLabel(season: number): string {
  const y = season % 100;
  const next = (y + 1) % 100;
  return `SEASON ${String(y).padStart(2, '0')}/${String(next).padStart(2, '0')}`;
}

function leagueTitleParts(name: string): {lineA: string; lineB: string} {
  const u = name.toUpperCase().trim();
  const parts = u.split(/\s+/).filter(Boolean);
  if (parts.length >= 2) {
    return {lineA: parts[0], lineB: parts.slice(1).join(' ')};
  }
  return {lineA: u, lineB: ''};
}

interface TableScreenProps {
  match: Match;
  matchScrollHandler?: MatchDetailsScrollHandler;
}

export const TableScreen: React.FC<TableScreenProps> = ({
  match,
  matchScrollHandler,
}) => {
  const matchScroll = useMatchDetailsScroll();
  const onScroll = matchScrollHandler ?? matchScroll?.scrollHandler;
  const [standings, setStandings] = useState<Standing[][]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedGroups, setExpandedGroups] = useState<Record<number, boolean>>(
    {},
  );

  useEffect(() => {
    if (!match || !match.league || !match.league.id) {
      console.warn('Skipping fetch: invalid match or league id', match);
      return;
    }
    const loadStandings = async () => {
      try {
        setLoading(true);
        const {id, season} = match.league;
        const data = await fetchStandings(id, season);
        setStandings(data as unknown as Standing[][]);
        setError(null);
      } catch (err) {
        setError('Failed to load standings');
        console.log('Error loading standings:', err);
      } finally {
        setLoading(false);
      }
    };

    loadStandings();
  }, [match]);

  const insightCopy = useMemo(() => {
    const flat = standings.flat();
    if (flat.length === 0) return null;
    const sorted = [...flat].sort((a, b) => {
      if (b.points !== a.points) return b.points - a.points;
      return goalDiff(b) - goalDiff(a);
    });
    const leader = sorted[0];
    if (!leader) return null;
    const gd = goalDiff(leader);
    return {
      leaderName: leader.team.name,
      pts: leader.points,
      gd,
      played: leader.all.played,
    };
  }, [standings]);

  if (loading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color={MATCH_CENTER_LIME} />
        <Text style={styles.loadingText}>Loading table...</Text>
      </View>
    );
  }

  if (error) {
    return (
      <View style={styles.centered}>
        <Text style={styles.errorText}>{error}</Text>
      </View>
    );
  }

  const toggleGroup = (idx: number) => {
    setExpandedGroups(prev => ({...prev, [idx]: !prev[idx]}));
  };

  return (
    <Animated.ScrollView
      style={styles.container}
      contentContainerStyle={styles.scrollContent}
      onScroll={onScroll}
      scrollEventThrottle={16}
      nestedScrollEnabled
      showsVerticalScrollIndicator={false}>
      {Array.isArray(standings) &&
        standings.map((group: Standing[], groupIdx: number) => {
          const homeName = normName(match.teams.home.name);
          const focusRow = group.find(s => normName(s.team.name) === homeName);
          const focusStanding =
            focusRow ??
            group.find(
              s => normName(s.team.name) === normName(match.teams.away.name),
            );

          const isExpanded = expandedGroups[groupIdx] === true;
          const displayRows = isExpanded ? group : group.slice(0, PREVIEW_ROWS);
          const hasMore = group.length > PREVIEW_ROWS;

          return (
            <View key={groupIdx} style={styles.groupWrap}>
              {group[0]?.group ? (
                <Text style={styles.groupLabel}>{group[0].group}</Text>
              ) : null}

              <View style={styles.tableCard}>
                <View style={styles.tableHeader}>
                  <Text style={[styles.th, styles.thRank]}>#</Text>
                  <Text style={[styles.th, styles.thTeam]}>TEAM</Text>
                  <Text style={[styles.th, styles.thStat]}>P</Text>
                  <Text style={[styles.th, styles.thStat]}>W</Text>
                  <Text style={[styles.th, styles.thStat]}>D</Text>
                  <Text style={[styles.th, styles.thStat]}>L</Text>
                  <Text style={[styles.th, styles.thGd]}>GD</Text>
                  <Text style={[styles.th, styles.thPts]}>PTS</Text>
                </View>

                {displayRows.map((standing: Standing) => {
                  const gdVal = goalDiff(standing);
                  const isFocus =
                    focusStanding != null &&
                    standing.team.id === focusStanding.team.id;
                  const topFour = standing.rank <= 4;

                  return (
                    <View
                      key={`${standing.rank}-${standing.team.id}`}
                      style={[
                        styles.tableRow,
                        isFocus && styles.tableRowFocus,
                      ]}>
                      {isFocus ? <View style={styles.focusRail} /> : null}
                      <Text
                        style={[
                          styles.td,
                          styles.tdRank,
                          topFour ? styles.rankTop : styles.rankRest,
                        ]}>
                        {String(standing.rank).padStart(2, '0')}
                      </Text>
                      <View style={styles.tdTeam}>
                        <Image
                          source={{uri: standing.team.logo}}
                          style={styles.teamLogo}
                        />
                        <View style={styles.teamNameCol}>
                          <Text
                            style={[
                              styles.teamName,
                              isFocus && styles.teamNameFocus,
                            ]}
                            numberOfLines={2}>
                            {standing.team.name.toUpperCase()}
                          </Text>
                          {isFocus ? (
                            <Text style={styles.yourTeam}>YOUR TEAM</Text>
                          ) : null}
                        </View>
                      </View>
                      <Text
                        style={[
                          styles.td,
                          styles.tdStat,
                          isFocus && styles.statFocus,
                        ]}>
                        {standing.all.played}
                      </Text>
                      <Text
                        style={[
                          styles.td,
                          styles.tdStat,
                          isFocus && styles.statFocus,
                        ]}>
                        {standing.all.win}
                      </Text>
                      <Text
                        style={[
                          styles.td,
                          styles.tdStat,
                          isFocus && styles.statFocus,
                        ]}>
                        {standing.all.draw}
                      </Text>
                      <Text
                        style={[
                          styles.td,
                          styles.tdStat,
                          isFocus && styles.statFocus,
                        ]}>
                        {standing.all.lose}
                      </Text>
                      <Text style={[styles.td, styles.tdGd]}>
                        {formatGD(gdVal)}
                      </Text>
                      <Text
                        style={[
                          styles.td,
                          styles.tdPts,
                          isFocus && styles.ptsFocus,
                        ]}>
                        {standing.points}
                      </Text>
                    </View>
                  );
                })}
              </View>

              {hasMore ? (
                <Pressable
                  style={styles.showMore}
                  onPress={() => toggleGroup(groupIdx)}
                  hitSlop={8}>
                  <Text style={styles.showMoreText}>
                    {isExpanded ? 'SHOW LESS' : 'SHOW FULL TABLE'}
                  </Text>
                  <Ionicons
                    name={isExpanded ? 'chevron-up' : 'chevron-down'}
                    size={18}
                    color={MATCH_CENTER_LIME}
                  />
                </Pressable>
              ) : null}
            </View>
          );
        })}

      {/* Statistical insight */}
      {insightCopy ? (
        <View style={styles.insightCard}>
          <View style={styles.insightRail} />
          <View style={styles.insightBody}>
            <View style={styles.insightHeader}>
              <View style={styles.insightIconBox}>
                <Ionicons
                  name="analytics"
                  size={18}
                  color={MATCH_CENTER_CYAN}
                />
              </View>
              <Text style={styles.insightTitle}>STATISTICAL INSIGHT</Text>
            </View>
            <Text style={styles.insightBodyText}>
              <Text style={styles.insightMuted}>
                Current leaders{' '}
                <Text style={styles.insightCyan}>{insightCopy.leaderName}</Text>{' '}
                sit on{' '}
                <Text style={styles.insightCyan}>{insightCopy.pts} pts</Text>{' '}
                from{' '}
                <Text style={styles.insightCyan}>
                  {insightCopy.played} played
                </Text>
                , with a goal difference of{' '}
                <Text style={styles.insightLime}>
                  {formatGD(insightCopy.gd)}
                </Text>
                . Form and fixture difficulty can still shift the table before
                the final whistle.
              </Text>
            </Text>
          </View>
        </View>
      ) : null}

      <View style={styles.bottomSpacer} />
    </Animated.ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: MATCH_CENTER_BG,
  },
  scrollContent: {
    paddingHorizontal: PAGE,
    paddingTop: 12,
    paddingBottom: 28,
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: MATCH_CENTER_BG,
    paddingHorizontal: 24,
  },
  loadingText: {
    marginTop: 10,
    fontSize: 14,
    color: MATCH_CENTER_MUTED,
  },
  errorText: {
    color: '#ff6b6b',
    fontSize: 15,
    textAlign: 'center',
  },
  heroCard: {
    backgroundColor: MATCH_CENTER_CARD,
    borderRadius: 14,
    padding: 16,
    marginBottom: 14,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.08)',
    overflow: 'hidden',
  },
  heroTrophyWatermark: {
    position: 'absolute',
    right: -16,
    top: 8,
    opacity: 0.07,
  },
  heroHeaderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 10,
  },
  heroKicker: {
    fontSize: 10,
    fontWeight: '900',
    letterSpacing: 1.2,
    color: MATCH_CENTER_MUTED,
  },
  heroLeagueMain: {
    marginBottom: 12,
  },
  heroLeagueWhite: {
    fontSize: 22,
    fontWeight: '900',
    letterSpacing: 0.5,
    color: MATCH_CENTER_TEXT,
  },
  heroLeagueLime: {
    fontSize: 22,
    fontWeight: '900',
    letterSpacing: 0.5,
    color: MATCH_CENTER_LIME,
  },
  heroFooterRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  heroSeason: {
    fontSize: 11,
    fontWeight: '700',
    color: MATCH_CENTER_MUTED,
    letterSpacing: 0.5,
  },
  livePill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  liveDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
    backgroundColor: MATCH_CENTER_LIME,
  },
  liveText: {
    fontSize: 10,
    fontWeight: '800',
    letterSpacing: 0.6,
    color: MATCH_CENTER_MUTED,
  },
  groupWrap: {
    marginBottom: 8,
  },
  groupLabel: {
    fontSize: 12,
    fontWeight: '800',
    color: MATCH_CENTER_MUTED,
    marginBottom: 8,
    letterSpacing: 0.5,
  },
  tableCard: {
    backgroundColor: MATCH_CENTER_CARD,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
    overflow: 'hidden',
  },
  tableHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 10,
    paddingHorizontal: 8,
    borderBottomWidth: 1,
    borderBottomColor: 'rgba(255,255,255,0.08)',
    backgroundColor: 'rgba(0,0,0,0.25)',
  },
  th: {
    fontSize: 9,
    fontWeight: '800',
    color: MATCH_CENTER_MUTED,
    textAlign: 'center',
    letterSpacing: 0.3,
  },
  thRank: {
    width: 28,
  },
  thTeam: {
    flex: 1,
    textAlign: 'left',
    paddingLeft: 4,
  },
  thStat: {
    width: 20,
  },
  thGd: {
    width: 30,
  },
  thPts: {
    width: 28,
  },
  tableRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 10,
    paddingHorizontal: 8,
    borderBottomWidth: 1,
    borderBottomColor: 'rgba(255,255,255,0.06)',
    position: 'relative',
  },
  tableRowFocus: {
    backgroundColor: 'rgba(223,255,0,0.06)',
  },
  focusRail: {
    position: 'absolute',
    left: 0,
    top: 0,
    bottom: 0,
    width: 3,
    backgroundColor: MATCH_CENTER_LIME,
  },
  td: {
    fontSize: 10,
    fontWeight: '700',
    color: MATCH_CENTER_TEXT,
    textAlign: 'center',
  },
  tdRank: {
    width: 28,
    fontWeight: '900',
  },
  rankTop: {
    color: MATCH_CENTER_LIME,
  },
  rankRest: {
    color: MATCH_CENTER_TEXT,
  },
  tdTeam: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    paddingLeft: 4,
    minWidth: 0,
  },
  teamLogo: {
    width: 22,
    height: 22,
    marginRight: 8,
    borderRadius: 4,
  },
  teamNameCol: {
    flex: 1,
    minWidth: 0,
  },
  teamName: {
    fontSize: 10,
    fontWeight: '800',
    color: MATCH_CENTER_TEXT,
    letterSpacing: 0.2,
  },
  teamNameFocus: {
    color: MATCH_CENTER_LIME,
  },
  yourTeam: {
    fontSize: 8,
    fontWeight: '900',
    color: MATCH_CENTER_LIME,
    marginTop: 2,
    letterSpacing: 0.4,
  },
  statFocus: {
    color: MATCH_CENTER_LIME,
  },
  tdStat: {
    width: 20,
  },
  tdGd: {
    width: 30,
    fontWeight: '800',
    color: MATCH_CENTER_CYAN,
  },
  tdPts: {
    width: 28,
    fontWeight: '900',
  },
  ptsFocus: {
    color: MATCH_CENTER_LIME,
  },
  showMore: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    paddingVertical: 14,
  },
  showMoreText: {
    fontSize: 11,
    fontWeight: '900',
    letterSpacing: 0.8,
    color: MATCH_CENTER_LIME,
  },
  insightCard: {
    flexDirection: 'row',
    backgroundColor: MATCH_CENTER_CARD,
    borderRadius: 12,
    marginTop: 8,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.06)',
    overflow: 'hidden',
  },
  insightRail: {
    width: 4,
    backgroundColor: MATCH_CENTER_CYAN,
  },
  insightBody: {
    flex: 1,
    padding: 14,
  },
  insightHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    marginBottom: 10,
  },
  insightIconBox: {
    width: 36,
    height: 36,
    borderRadius: 8,
    backgroundColor: 'rgba(0,229,255,0.1)',
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    borderColor: 'rgba(0,229,255,0.2)',
  },
  insightTitle: {
    fontSize: 11,
    fontWeight: '900',
    letterSpacing: 0.8,
    color: MATCH_CENTER_TEXT,
  },
  insightBodyText: {
    fontSize: 13,
    lineHeight: 20,
  },
  insightMuted: {
    color: MATCH_CENTER_MUTED,
  },
  insightCyan: {
    color: MATCH_CENTER_CYAN,
    fontWeight: '800',
  },
  insightLime: {
    color: MATCH_CENTER_LIME,
    fontWeight: '800',
  },
  bottomSpacer: {
    height: 8,
  },
});
