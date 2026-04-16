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
};

/**
 * Horizontal league badges with white inner tile (matches + news feeds).
 */
const ALL_KEY = -1;

export function LeagueHorizontalStrip({
  selectedLeague,
  onSelect,
  includeAllOption = false,
  accentColor,
  mutedColor,
}: Props) {
  const scrollRef = useRef<ScrollView>(null);
  /** Content-relative x for each league id (and ALL_KEY when includeAllOption). */
  const itemXRef = useRef<Record<number, number>>({});

  const scrollSelectedIntoView = useCallback((itemX: number) => {
    const pad = 48;
    scrollRef.current?.scrollTo({
      x: Math.max(0, itemX - pad),
      animated: true,
    });
  }, []);

  useEffect(() => {
    const targetId =
      selectedLeague === null
        ? includeAllOption
          ? ALL_KEY
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
  }, [selectedLeague, includeAllOption, scrollSelectedIntoView]);

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
              itemXRef.current[ALL_KEY] = x;
              if (selectedLeague === null) {
                scrollSelectedIntoView(x);
              }
            }}>
            <View
              style={[
                styles.iconWrap,
                selectedLeague === null && [
                  styles.iconWrapSelected,
                  {borderColor: accentColor},
                ],
              ]}>
              <View style={styles.iconInner}>
                <Text style={styles.allGlyph}>∞</Text>
              </View>
            </View>
            <Text
              style={[
                styles.label,
                {color: mutedColor},
                selectedLeague === null && {color: accentColor},
              ]}
              numberOfLines={1}>
              ALL
            </Text>
            {selectedLeague === null ? (
              <View style={[styles.underline, {backgroundColor: accentColor}]} />
            ) : (
              <View style={styles.underlinePlaceholder} />
            )}
          </TouchableOpacity>
        )}
        {LEAGUES.map(league => {
          const isSelected =
            selectedLeague !== null && selectedLeague.id === league.id;
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
              <View
                style={[
                  styles.iconWrap,
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
  iconWrap: {
    width: 44,
    height: 44,
    borderRadius: 8,
    backgroundColor: 'rgba(255,255,255,0.06)',
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.12)',
    padding: 3,
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
