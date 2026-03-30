import React from 'react';
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
export function LeagueHorizontalStrip({
  selectedLeague,
  onSelect,
  includeAllOption = false,
  accentColor,
  mutedColor,
}: Props) {
  return (
    <View style={styles.strip}>
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.content}>
        {includeAllOption && (
          <TouchableOpacity
            style={styles.item}
            onPress={() => onSelect(null)}
            activeOpacity={0.85}>
            <View style={styles.iconWrap}>
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
              activeOpacity={0.85}>
              <View style={styles.iconWrap}>
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
