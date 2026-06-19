import React, {useCallback, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  Image,
  type StyleProp,
  type ViewStyle,
} from 'react-native';
import {useNavigation} from '@react-navigation/native';
import {Ionicons} from '@expo/vector-icons';
import {LinearGradient} from 'expo-linear-gradient';
import type {NavigationProp} from '../types/navigation';
import {useMediaShorts} from '../hooks/useMediaShorts';
import type {MediaOutletShorts} from '../services/mediaShortsApi';
import {
  NEWS_ACCENT,
  NEWS_CARD,
  NEWS_CYAN,
  NEWS_MUTED,
  NEWS_TEXT,
} from '../constants/newsUi';

const STORY_RING_SIZE = 72;

export type MediaStoryRingsTheme = {
  accent: string;
  cyan: string;
  text: string;
  muted: string;
  card: string;
};

const DEFAULT_THEME: MediaStoryRingsTheme = {
  accent: NEWS_ACCENT,
  cyan: NEWS_CYAN,
  text: NEWS_TEXT,
  muted: NEWS_MUTED,
  card: NEWS_CARD,
};

type MediaStoryRingsProps = {
  enabled?: boolean;
  theme?: MediaStoryRingsTheme;
  contentContainerStyle?: StyleProp<ViewStyle>;
};

export function MediaStoryRings({
  enabled = true,
  theme = DEFAULT_THEME,
  contentContainerStyle,
}: MediaStoryRingsProps) {
  const navigation = useNavigation<NavigationProp>();
  const {data: mediaShortsData} = useMediaShorts();
  const mediaOutlets = mediaShortsData?.outlets ?? [];
  const [failedStoryThumbs, setFailedStoryThumbs] = useState<Set<string>>(
    new Set(),
  );

  const handleStoryThumbError = useCallback((lookupKey: string) => {
    setFailedStoryThumbs(prev => new Set(prev).add(lookupKey));
  }, []);

  const onMediaStoryPress = useCallback(
    (outlet: MediaOutletShorts) => {
      navigation.navigate('MatchTeamShorts', {
        shorts: outlet.shorts ?? [],
        teamDisplayName: outlet.display_name,
      });
    },
    [navigation],
  );

  if (!enabled || mediaOutlets.length === 0) {
    return null;
  }

  const renderStoryRing = (outlet: MediaOutletShorts) => {
    const thumbOk =
      Boolean(outlet.thumbnail_url) &&
      !failedStoryThumbs.has(outlet.lookup_key);
    const active = outlet.has_shorts;

    const inner = thumbOk ? (
      <Image
        source={{uri: outlet.thumbnail_url}}
        style={styles.storyThumb}
        resizeMode="cover"
        onError={() => handleStoryThumbError(outlet.lookup_key)}
      />
    ) : (
      <View style={[styles.storyInner, {backgroundColor: theme.card}]}>
        <Ionicons
          name="logo-youtube"
          size={28}
          color={active ? theme.text : theme.muted}
        />
      </View>
    );

    return (
      <TouchableOpacity
        key={outlet.lookup_key}
        style={styles.storyItem}
        onPress={() => onMediaStoryPress(outlet)}
        activeOpacity={0.88}>
        {active ? (
          <LinearGradient
            colors={[theme.accent, theme.cyan]}
            start={{x: 0, y: 0}}
            end={{x: 1, y: 1}}
            style={styles.storyGradient}>
            <View style={styles.storyInnerClip}>{inner}</View>
          </LinearGradient>
        ) : (
          <View style={[styles.storyGradient, styles.storyGradientMuted]}>
            <View style={styles.storyInnerClip}>{inner}</View>
          </View>
        )}
        <Text
          style={[
            styles.storyLabel,
            {color: theme.text},
            !active && {color: theme.muted},
          ]}
          numberOfLines={2}>
          {outlet.display_name}
        </Text>
      </TouchableOpacity>
    );
  };

  return (
    <ScrollView
      horizontal
      showsHorizontalScrollIndicator={false}
      contentContainerStyle={[styles.storyRow, contentContainerStyle]}>
      {mediaOutlets.map(renderStoryRing)}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  storyRow: {
    paddingTop: 8,
    paddingBottom: 12,
    flexDirection: 'row',
    alignItems: 'flex-start',
  },
  storyItem: {
    alignItems: 'center',
    marginRight: 14,
    width: 78,
  },
  storyGradient: {
    borderRadius: 18,
    padding: 3,
  },
  storyGradientMuted: {
    backgroundColor: 'rgba(255,255,255,0.12)',
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.18)',
  },
  storyInnerClip: {
    width: STORY_RING_SIZE,
    height: STORY_RING_SIZE,
    borderRadius: 15,
    overflow: 'hidden',
  },
  storyInner: {
    width: STORY_RING_SIZE,
    height: STORY_RING_SIZE,
    borderRadius: 15,
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
  },
  storyThumb: {
    width: STORY_RING_SIZE,
    height: STORY_RING_SIZE,
  },
  storyLabel: {
    marginTop: 8,
    fontSize: 9,
    fontWeight: '800',
    letterSpacing: 0.4,
    textAlign: 'center',
  },
});
