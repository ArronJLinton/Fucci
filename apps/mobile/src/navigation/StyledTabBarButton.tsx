import React from 'react';
import {
  Pressable,
  type PressableStateCallbackType,
  StyleSheet,
  View,
  type StyleProp,
  type ViewStyle,
} from 'react-native';
import type {BottomTabBarButtonProps} from '@react-navigation/bottom-tabs';

/** Matches debates UI accent */
export const TAB_LIME = '#C6FF00';

type Variant = 'light' | 'dark';

export type StyledTabBarButtonProps = BottomTabBarButtonProps & {
  variant: Variant;
};

function resolveStyle(
  style:
    | StyleProp<ViewStyle>
    | ((state: PressableStateCallbackType) => StyleProp<ViewStyle>)
    | undefined,
  pressed: boolean,
): StyleProp<ViewStyle> {
  if (typeof style === 'function') {
    return style({pressed});
  }
  return style;
}

/**
 * Focused tab: rounded pill, subtle fill + olive border (dark) or grey (light).
 * Unfocused: no chrome; icon/label colors come from tabBar tint props.
 */
export const StyledTabBarButton = React.forwardRef<
  React.ComponentRef<typeof Pressable>,
  StyledTabBarButtonProps
>(function StyledTabBarButton(
  {children, style, variant, ...pressableProps},
  ref,
) {
  const selected = pressableProps.accessibilityState?.selected;

  return (
    <Pressable
      ref={ref}
      {...pressableProps}
      style={state => [
        styles.hit,
        resolveStyle(style, state.pressed),
      ]}>
      <View
        style={[
          styles.pill,
          variant === 'dark'
            ? selected
              ? styles.pillDarkFocused
              : styles.pillDarkIdle
            : selected
              ? styles.pillLightFocused
              : styles.pillLightIdle,
        ]}>
        {children}
      </View>
    </Pressable>
  );
});

const styles = StyleSheet.create({
  hit: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'stretch',
  },
  pill: {
    flex: 1,
    marginHorizontal: 2,
    marginVertical: 4,
    borderRadius: 10,
    justifyContent: 'center',
    alignItems: 'center',
    paddingVertical: 4,
    paddingHorizontal: 4,
  },
  pillDarkIdle: {
    backgroundColor: 'transparent',
    borderWidth: 0,
  },
  pillDarkFocused: {
    backgroundColor: 'rgba(198, 255, 0, 0.08)',
    borderWidth: 1,
    borderColor: '#3D4F38',
  },
  pillLightIdle: {
    backgroundColor: 'transparent',
    borderWidth: 0,
  },
  pillLightFocused: {
    backgroundColor: '#E8E8ED',
    borderWidth: 1,
    borderColor: '#C7C7CC',
  },
});
