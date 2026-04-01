import React from 'react';
import {
  Image,
  type ImageSourcePropType,
  StyleSheet,
  Text,
  View,
  type ViewStyle,
} from 'react-native';

/** 1-based indices match `assets/trait_hex/trait_{n}.png` (reference grid #1–#14). */
const TRAIT_CODE_TO_SHEET_INDEX: Record<string, number> = {
  SPEED_DRIBBLER: 1,
  PLAYMAKER: 3,
  LONG_SHOT_TAKER: 8,
  OUTSIDE_FOOT_SHOT: 6,
  POWER_HEADER: 7,
  FLAIR: 9,
  FINESSE_SHOT: 10,
  LEADERSHIP: 11,
  POWER_FREE_KICK: 12,
};

const TRAIT_HEX_SOURCES: ImageSourcePropType[] = [
  require('../../assets/trait_hex/trait_1.png'),
  require('../../assets/trait_hex/trait_2.png'),
  require('../../assets/trait_hex/trait_3.png'),
  require('../../assets/trait_hex/trait_4.png'),
  require('../../assets/trait_hex/trait_5.png'),
  require('../../assets/trait_hex/trait_6.png'),
  require('../../assets/trait_hex/trait_7.png'),
  require('../../assets/trait_hex/trait_8.png'),
  require('../../assets/trait_hex/trait_9.png'),
  require('../../assets/trait_hex/trait_10.png'),
  require('../../assets/trait_hex/trait_11.png'),
  require('../../assets/trait_hex/trait_12.png'),
  require('../../assets/trait_hex/trait_13.png'),
  require('../../assets/trait_hex/trait_14.png'),
];

export const PLAYER_TRAIT_LABELS: Record<string, string> = {
  LEADERSHIP: 'Leadership',
  FINESSE_SHOT: 'Finesse Shot',
  PLAYMAKER: 'Playmaker',
  SPEED_DRIBBLER: 'Speed Dribbler',
  LONG_SHOT_TAKER: 'Long Shot Taker',
  OUTSIDE_FOOT_SHOT: 'Outside Foot Shot',
  POWER_HEADER: 'Power Header',
  FLAIR: 'Flair',
  POWER_FREE_KICK: 'Power Free Kick',
};

export function traitSheetIndexForCode(code: string): number | null {
  const n = TRAIT_CODE_TO_SHEET_INDEX[code];
  return n ?? null;
}

export function traitHexSourceForCode(code: string): ImageSourcePropType | null {
  const n = traitSheetIndexForCode(code);
  if (n == null) return null;
  return TRAIT_HEX_SOURCES[n - 1] ?? null;
}

const GREEN = '#4ade80';

export function TraitHexImage({
  code,
  size = 50,
  style,
}: {
  code: string;
  size?: number;
  style?: ViewStyle;
}) {
  const src = traitHexSourceForCode(code);
  if (!src) {
    return <View style={[{width: size, height: size}, style]} />;
  }
  return (
    <Image
      source={src}
      style={[{width: size, height: size}, styles.hexImg, style]}
      resizeMode="contain"
      accessibilityIgnoresInvertColors
    />
  );
}

export function PlayerTraitBadgeRow({
  code,
  shellStyle,
}: {
  code: string;
  shellStyle?: ViewStyle;
}) {
  const label = PLAYER_TRAIT_LABELS[code] ?? code.replace(/_/g, ' ');
  return (
    <View style={[styles.badgeShell, shellStyle]}>
      <TraitHexImage code={code} size={50} />
      <Text style={styles.traitNameText}>{label.toUpperCase()}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  hexImg: {},
  badgeShell: {
    flexDirection: 'column',
    alignItems: 'center',
    gap: 8,
  },
  traitNameText: {
    fontSize: 12,
    fontWeight: '800',
    color: GREEN,
    letterSpacing: 0.35,
    textAlign: 'center',
    alignSelf: 'stretch',
  },
});
