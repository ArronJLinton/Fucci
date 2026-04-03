import type {PlayerProfile} from '../types/playerProfile';

/** Default rating for core attributes (40–99) before user or position-specific tuning. */
export const DEFAULT_CORE_RATING = 50;

/**
 * Base core attributes used on the player profile and compare screens.
 * Defaults are neutral (50) for all positions; users tune via sliders.
 */
export function coreAttrsForPosition(
  pos: PlayerProfile['position'] | null,
): {
  speed: number;
  shooting: number;
  passing: number;
  physical: number;
  stamina: number;
} {
  void pos;
  return {
    speed: DEFAULT_CORE_RATING,
    shooting: DEFAULT_CORE_RATING,
    passing: DEFAULT_CORE_RATING,
    physical: DEFAULT_CORE_RATING,
    stamina: DEFAULT_CORE_RATING,
  };
}

export function dribblingDefendingForPosition(
  pos: PlayerProfile['position'] | null,
): {dribbling: number; defending: number} {
  void pos;
  return {
    dribbling: DEFAULT_CORE_RATING,
    defending: DEFAULT_CORE_RATING,
  };
}
