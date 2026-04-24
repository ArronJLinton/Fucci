import React, {useCallback, useEffect, useRef} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  Image,
} from 'react-native';
import {
  LEAGUES,
  NEWS_STRIP_ALL_LEAGUE_ID,
  leagueStripLabel,
  type League,
} from '../constants/leagues';

type Props = {
  selectedLeague: League | null;
  onSelect: (league: League | null) => void;
  /** When true, first pill is "ALL" (clears league filter). */
  includeAllOption?: boolean;
  accentColor: string;
  mutedColor: string;
  /** When true for a league id, draw a lime-style ring around the badge (Snapchat stories available). */
  snapStoryRingByLeagueId?: Partial<Record<number, boolean>>;
  /** Border + glow color for `snapStoryRingByLeagueId`; defaults to `accentColor`. */
  snapRingColor?: string;
  /** When false, no pill underline / lime border on the active league (e.g. News tap opens stories only). */
  showSelectionHighlight?: boolean;
};

/**
 * Horizontal league badges with white inner tile (matches + news feeds).
 */
export function LeagueHorizontalStrip({
  selectedLeague,
  onSelect,
  includeAllOption = false,
  accentColor,
  mutedColor,
  snapStoryRingByLeagueId,
  snapRingColor,
  showSelectionHighlight = true,
}: Props) {
  const ringTint = snapRingColor ?? accentColor;
  const scrollRef = useRef<ScrollView>(null);
  /** Content-relative x for each league id (and `NEWS_STRIP_ALL_LEAGUE_ID` when includeAllOption). */
  const itemXRef = useRef<Record<number, number>>({});

  const showSnapRingAll =
    Boolean(includeAllOption) &&
    Boolean(snapStoryRingByLeagueId?.[NEWS_STRIP_ALL_LEAGUE_ID]);

  const allHighlighted =
    showSelectionHighlight &&
    includeAllOption &&
    selectedLeague === null;

  const scrollSelectedIntoView = useCallback((itemX: number) => {
    const pad = 48;
    scrollRef.current?.scrollTo({
      x: Math.max(0, itemX - pad),
      animated: true,
    });
  }, []);

  useEffect(() => {
    if (!showSelectionHighlight) {
      return;
    }
    const targetId =
      selectedLeague === null
        ? includeAllOption
          ? NEWS_STRIP_ALL_LEAGUE_ID
          : null
        : selectedLeague.id;
    if (targetId === null) {
      return;
    }

    let cancelled = false;

    const tryScroll = (): boolean => {
      if (cancelled) {
        return true;
      }
      const x = itemXRef.current[targetId];
      if (x != null) {
        scrollSelectedIntoView(x);
        return true;
      }
      return false;
    };

    if (tryScroll()) {
      return () => {
        cancelled = true;
      };
    }

    const t1 = setTimeout(() => {
      tryScroll();
    }, 50);
    const t2 = setTimeout(() => {
      tryScroll();
    }, 200);
    return () => {
      cancelled = true;
      clearTimeout(t1);
      clearTimeout(t2);
    };
  }, [selectedLeague, includeAllOption, scrollSelectedIntoView, showSelectionHighlight]);

  return (
    <View style={styles.strip}>
      <ScrollView
        ref={scrollRef}
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.content}>
        {includeAllOption && (
          <TouchableOpacity
            style={styles.item}
            onPress={() => onSelect(null)}
            activeOpacity={0.85}
            onLayout={e => {
              const x = e.nativeEvent.layout.x;
              itemXRef.current[NEWS_STRIP_ALL_LEAGUE_ID] = x;
              if (allHighlighted) {
                scrollSelectedIntoView(x);
              }
            }}>
            {showSnapRingAll ? (
              <View
                style={[
                  styles.snapRingOuter,
                  {borderColor: ringTint, shadowColor: ringTint},
                ]}>
                <View
                  style={[
                    styles.iconWrap,
                    allHighlighted && [
                      styles.iconWrapSelected,
                      {borderColor: accentColor},
                    ],
                  ]}>
                  <View style={styles.iconInner}>
                    <Text style={styles.allGlyph}>∞</Text>
                  </View>
                </View>
              </View>
            ) : (
              <View
                style={[
                  styles.iconWrap,
                  styles.iconWrapMarginBottom,
                  allHighlighted && [
                    styles.iconWrapSelected,
                    {borderColor: accentColor},
                  ],
                ]}>
                <View style={styles.iconInner}>
                  <Text style={styles.allGlyph}>∞</Text>
                </View>
              </View>
            )}
            <Text
              style={[
                styles.label,
                {color: mutedColor},
                allHighlighted && {color: accentColor},
              ]}
              numberOfLines={1}>
              ALL
            </Text>
            {allHighlighted ? (
              <View style={[styles.underline, {backgroundColor: accentColor}]} />
            ) : (
              <View style={styles.underlinePlaceholder} />
            )}
          </TouchableOpacity>
        )}
        {LEAGUES.map(league => {
          const isSelected =
            showSelectionHighlight &&
            selectedLeague !== null &&
            selectedLeague.id === league.id;
          const showSnapRing = Boolean(snapStoryRingByLeagueId?.[league.id]);
          const logoBlock = (
            <View
              style={[
                styles.iconWrap,
                !showSnapRing && styles.iconWrapMarginBottom,
                isSelected && [
                  styles.iconWrapSelected,
                  {borderColor: accentColor},
                ],
              ]}>
              <View style={styles.iconInner}>
                {league.logo ? (
                  <Image
                    source={{uri: league.logo}}
                    style={styles.iconImg}
                    resizeMode="contain"
                  />
                ) : (
                  <Text style={styles.fallback}>
                    {leagueStripLabel(league.name).slice(0, 2)}
                  </Text>
                )}
              </View>
            </View>
          );
          return (
            <TouchableOpacity
              key={league.id}
              style={styles.item}
              onPress={() => onSelect(league)}
              activeOpacity={0.85}
              onLayout={e => {
                const x = e.nativeEvent.layout.x;
                itemXRef.current[league.id] = x;
                if (isSelected) {
                  scrollSelectedIntoView(x);
                }
              }}>
              {showSnapRing ? (
                <View
                  style={[
                    styles.snapRingOuter,
                    {borderColor: ringTint, shadowColor: ringTint},
                  ]}>
                  {logoBlock}
                </View>
              ) : (
                logoBlock
              )}
              <Text
                style={[
                  styles.label,
                  {color: mutedColor},
                  isSelected && {color: accentColor},
                ]}
                numberOfLines={1}>
                {leagueStripLabel(league.name)}
              </Text>
              {isSelected ? (
                <View style={[styles.underline, {backgroundColor: accentColor}]} />
              ) : (
                <View style={styles.underlinePlaceholder} />
              )}
            </TouchableOpacity>
          );
        })}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  strip: {
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: 'rgba(255,255,255,0.06)',
  },
  content: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    paddingHorizontal: 12,
  },
  item: {
    alignItems: 'center',
    marginHorizontal: 8,
    minWidth: 64,
  },
  snapRingOuter: {
    padding: 2,
    borderRadius: 10,
    borderWidth: 2,
    marginBottom: 6,
    alignSelf: 'center',
    shadowOffset: {width: 0, height: 0},
    shadowOpacity: 0.85,
    shadowRadius: 14,
    elevation: 12,
  },
  iconWrap: {
    width: 44,
    height: 44,
    borderRadius: 8,
    backgroundColor: 'rgba(255,255,255,0.06)',
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.12)',
    padding: 3,
  },
  iconWrapMarginBottom: {
    marginBottom: 6,
  },
  iconWrapSelected: {
    backgroundColor: 'rgba(198,255,0,0.14)',
    borderWidth: 2,
    shadowColor: '#C6FF00',
    shadowOffset: {width: 0, height: 0},
    shadowOpacity: 0.35,
    shadowRadius: 6,
    elevation: 4,
  },
  iconInner: {
    width: 38,
    height: 38,
    borderRadius: 6,
    backgroundColor: '#FFFFFF',
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
  },
  iconImg: {
    width: 30,
    height: 30,
  },
  fallback: {
    fontSize: 11,
    fontWeight: '800',
    color: '#0B0E14',
  },
  allGlyph: {
    fontSize: 18,
    fontWeight: '700',
    color: '#0B0E14',
  },
  label: {
    fontSize: 10,
    fontWeight: '700',
    letterSpacing: 0.6,
    textAlign: 'center',
  },
  underline: {
    marginTop: 6,
    height: 3,
    width: 32,
    borderRadius: 2,
  },
  underlinePlaceholder: {
    marginTop: 6,
    height: 3,
    width: 32,
  },
});
