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

/**
 * News screen "story rings" header (TOP GOALS / RUMOURS / MATCH DAY chips).
 *
 * When false the horizontal ring header is hidden on the News screen but all
 * supporting code (rings array, onStoryPress handler, styles) stays in place
 * so flipping the flag back to true re-enables it without a refactor.
 *
 * Currently disabled for the WC-only release: with the feed filtered to a
 * single competition the category chips don't meaningfully change the result
 * set. Re-enable once the news taxonomy / category filtering is reliable.
 */
export const NEWS_STORY_RINGS_ENABLED = false;
