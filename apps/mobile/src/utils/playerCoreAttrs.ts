/** Default rating for core attributes (40–99) before user tuning. */
export const DEFAULT_CORE_RATING = 50;

/**
 * Neutral defaults for the first “page” of core stats (speed, shooting, passing, physical, stamina).
 * Not position-specific; sliders apply on top.
 */
export function defaultCoreAttrs(): {
  speed: number;
  shooting: number;
  passing: number;
  physical: number;
  stamina: number;
} {
  return {
    speed: DEFAULT_CORE_RATING,
    shooting: DEFAULT_CORE_RATING,
    passing: DEFAULT_CORE_RATING,
    physical: DEFAULT_CORE_RATING,
    stamina: DEFAULT_CORE_RATING,
  };
}

/** Neutral defaults for dribbling and defending (second core page). */
export function defaultDribblingDefending(): {
  dribbling: number;
  defending: number;
} {
  return {
    dribbling: DEFAULT_CORE_RATING,
    defending: DEFAULT_CORE_RATING,
  };
}
