/**
 * Date-bucketing helpers for mapping the device's local calendar day onto the
 * UTC calendar days the upstream football API (API-Football) groups fixtures
 * by.
 *
 * Why this exists: API-Football's `/fixtures?date=YYYY-MM-DD` interprets the
 * date as UTC, so a fixture kicking off at 9 PM ET (= 01:00 UTC next day) is
 * returned for the *following* UTC date. If the mobile YESTERDAY/TODAY/TOMORROW
 * tabs naively trust that bucketing, late-night games show up under the wrong
 * tab for any user whose timezone is offset from UTC.
 *
 * Strategy: for each local-date tab, fetch the (one or two) UTC dates that
 * overlap it, then filter results client-side to the fixtures whose
 * `fixture.date` falls on the requested local date.
 */

/** Format a Date as YYYY-MM-DD using its UTC calendar components. */
export function formatUtcDate(d: Date): string {
  const y = d.getUTCFullYear();
  const m = String(d.getUTCMonth() + 1).padStart(2, '0');
  const day = String(d.getUTCDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

/** Format a Date as YYYY-MM-DD using its local calendar components. */
export function formatLocalDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

/**
 * Returns the unique UTC YYYY-MM-DD strings whose 24h UTC window overlaps the
 * given local calendar date.
 *
 * - Always returns 1 or 2 entries (in ascending order).
 * - For a viewer in UTC, returns exactly 1 entry equal to the local date.
 * - For any non-UTC offset, the local day straddles a UTC midnight and the
 *   result has 2 entries.
 *
 * Example (viewer in ET = UTC-4, summer):
 *   utcDateStringsForLocalDate(<midnight ET Tue Jun 16>)
 *     → ['2026-06-16', '2026-06-17']
 *   because ET Tue Jun 16 spans 04:00 UTC Jun 16 → 03:59 UTC Jun 17.
 *
 * Example (viewer in JST = UTC+9):
 *   utcDateStringsForLocalDate(<midnight JST Wed Jun 17>)
 *     → ['2026-06-16', '2026-06-17']
 *   because JST Wed Jun 17 spans 15:00 UTC Jun 16 → 14:59 UTC Jun 17.
 */
export function utcDateStringsForLocalDate(localDate: Date): string[] {
  const y = localDate.getFullYear();
  const m = localDate.getMonth();
  const d = localDate.getDate();

  // Construct local-midnight and local-end-of-day from the local components
  // (ignoring the input's H/M/S so callers can pass any moment within the day).
  const startOfLocalDay = new Date(y, m, d, 0, 0, 0, 0);
  const endOfLocalDay = new Date(y, m, d, 23, 59, 59, 999);

  const startUtc = formatUtcDate(startOfLocalDay);
  const endUtc = formatUtcDate(endOfLocalDay);

  return startUtc === endUtc ? [startUtc] : [startUtc, endUtc];
}

/**
 * True iff the given ISO-8601 (UTC-bearing) timestamp falls on `localDate`
 * when interpreted in the device's local timezone.
 *
 * The `localDate` argument is treated as a calendar day; its H/M/S are ignored.
 */
export function isOnLocalDate(isoUtc: string, localDate: Date): boolean {
  const fixture = new Date(isoUtc);
  if (Number.isNaN(fixture.getTime())) {
    return false;
  }
  return (
    fixture.getFullYear() === localDate.getFullYear() &&
    fixture.getMonth() === localDate.getMonth() &&
    fixture.getDate() === localDate.getDate()
  );
}
