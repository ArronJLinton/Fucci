/** Max “overall” level shown in profile/compare UI. */
const DISPLAY_LEVEL_MAX = 99;

/** Base offset before traits and profile completion contribute. */
const DISPLAY_LEVEL_BASE = 38;

/** Points added per equipped trait. */
const DISPLAY_LEVEL_PER_TRAIT = 9;

/** Weight in `round(completionPercent * factor)`; completionPercent is 0–100 (max bonus 20). */
const DISPLAY_LEVEL_COMPLETION_FACTOR = 0.2;

/**
 * Display “overall” level shown in profile/compare UI (traits + profile completion).
 *
 * @param traitsLen — Number of equipped traits (non-negative).
 * @param completionPercent — Profile completion on **0–100** (same scale as the “% complete” UI),
 *   not a 0–1 fraction.
 */
export function displayLevel(
  traitsLen: number,
  completionPercent: number,
): number {
  const fromCompletion = Math.round(
    completionPercent * DISPLAY_LEVEL_COMPLETION_FACTOR,
  );
  return Math.min(
    DISPLAY_LEVEL_MAX,
    DISPLAY_LEVEL_BASE + traitsLen * DISPLAY_LEVEL_PER_TRAIT + fromCompletion,
  );
}
