/**
 * Display “overall” level shown in profile/compare UI (traits + profile completion).
 * Capped at 99.
 */
export function displayLevel(
  traitsLen: number,
  completionPct: number,
): number {
  return Math.min(99, 38 + traitsLen * 9 + Math.round(completionPct * 0.2));
}
