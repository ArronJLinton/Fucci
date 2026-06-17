/**
 * Centralised feature flags.
 *
 * Keep these as plain module-scope booleans so callers can `import {FLAG}` and
 * the bundler dead-code-eliminates the unused branch in production builds.
 */

/**
 * Summer 2026 "World Cup only" mode.
 *
 * When true:
 *  - Matches screen shows ONLY World Cup fixtures (Yesterday / Today / Tomorrow tabs)
 *  - News feed is filtered to World Cup–related articles only
 *  - Debates feed is filtered to World Cup–related topics only
 *  - The horizontal league strip is hidden on Matches and News
 *
 * Rationale: Premier League / La Liga / Serie A / Bundesliga / Ligue 1 / UCL
 * seasons are all over until ~August 2026. Flip this to `false` when the new
 * club season starts.
 */
export const WORLD_CUP_ONLY_MODE = true;
