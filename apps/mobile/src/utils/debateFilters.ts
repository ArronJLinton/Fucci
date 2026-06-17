import {WORLD_CUP_ONLY_MODE} from '../config/featureFlags';
import type {DebateSummary} from '../types/debate';
import {worldCupKeywordMatch} from './newsFilters';

/** NEW DEBATES and MY ACTIVITY use `created_at` within this window (generation time). */
export const FEED_MAX_AGE_MS = 6 * 24 * 60 * 60 * 1000;

/**
 * Go encodes `time.Time` as RFC3339Nano. `Date.parse` often returns NaN when fractional
 * seconds go past milliseconds (e.g. `.123456789Z`), so every row was filtered out.
 */
export function parseFeedTimestampMs(iso: string | undefined): number | null {
  if (iso == null) {
    return null;
  }
  const s0 = String(iso).trim();
  if (!s0) {
    return null;
  }
  const tryParse = (s: string): number | null => {
    const n = Date.parse(s);
    return Number.isNaN(n) ? null : n;
  };
  let ms = tryParse(s0);
  if (ms != null) {
    return ms;
  }
  const withoutSubMsFrac = s0.replace(/(\.\d{3})\d+(?=[Zz]|[+-])/, '$1');
  ms = tryParse(withoutSubMsFrac);
  if (ms != null) {
    return ms;
  }
  const spaceToT = s0.includes('T') ? s0 : s0.replace(' ', 'T');
  return tryParse(spaceToT.replace(/(\.\d{3})\d+(?=[Zz]|[+-])/, '$1'));
}

function debateRowTimeMs(summary: DebateSummary): number | null {
  return (
    parseFeedTimestampMs(summary.created_at) ??
    parseFeedTimestampMs(summary.updated_at)
  );
}

export function debateGeneratedWithinPastSixDays(
  summary: DebateSummary,
  nowMs: number,
): boolean {
  const rowMs = debateRowTimeMs(summary);
  if (rowMs == null) {
    return true;
  }
  return nowMs - rowMs <= FEED_MAX_AGE_MS;
}

/**
 * Summer 2026 world-cup-only mode: keep a debate only if its headline,
 * description, or sourced headline mentions a World Cup-related keyword.
 */
export function debateIsWorldCupRelated(summary: DebateSummary): boolean {
  return (
    worldCupKeywordMatch(summary.headline) ||
    worldCupKeywordMatch(summary.description) ||
    worldCupKeywordMatch(summary.source_headline)
  );
}

export function debatePassesDiscoveryFilters(
  summary: DebateSummary,
  nowMs: number,
  worldCupOnlyMode = WORLD_CUP_ONLY_MODE,
): boolean {
  if (!debateGeneratedWithinPastSixDays(summary, nowMs)) {
    return false;
  }
  if (worldCupOnlyMode && !debateIsWorldCupRelated(summary)) {
    return false;
  }
  return true;
}

export function debatePassesActivityFilters(
  summary: DebateSummary,
  nowMs: number,
): boolean {
  return debateGeneratedWithinPastSixDays(summary, nowMs);
}
