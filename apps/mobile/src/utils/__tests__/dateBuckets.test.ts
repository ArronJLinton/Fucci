/// <reference types="jest" />

// Force Node's local timezone before any Date is constructed in this file.
// Mirrors the timezone of the user who reported the bug (Eastern Time) and
// covers both halves of the local day for fixtures whose UTC date differs.
process.env.TZ = 'America/New_York';

import {
  formatLocalDate,
  formatUtcDate,
  isOnLocalDate,
  utcDateStringsForLocalDate,
} from '../dateBuckets';

describe('formatUtcDate', () => {
  it('formats a UTC-anchored Date by its UTC components', () => {
    // 2026-06-17T01:00:00Z → "2026-06-17" regardless of local TZ.
    const d = new Date(Date.UTC(2026, 5, 17, 1, 0, 0));
    expect(formatUtcDate(d)).toBe('2026-06-17');
  });

  it('rolls over to the next UTC day for a local-ET 9 PM kickoff', () => {
    // 9 PM ET on Jun 16 = 01:00 UTC Jun 17 (DST: UTC-4).
    const kickoff = new Date('2026-06-17T01:00:00Z');
    expect(formatUtcDate(kickoff)).toBe('2026-06-17');
  });
});

describe('formatLocalDate', () => {
  it('formats a Date by its local (TZ=ET) components', () => {
    // 01:00 UTC Jun 17 == 21:00 ET Jun 16 (UTC-4 in summer).
    const kickoff = new Date('2026-06-17T01:00:00Z');
    expect(formatLocalDate(kickoff)).toBe('2026-06-16');
  });
});

describe('utcDateStringsForLocalDate (TZ=America/New_York)', () => {
  it('returns two UTC dates when the local day straddles UTC midnight', () => {
    // Local "Tue Jun 16 ET" runs 04:00 UTC Jun 16 → 03:59:59 UTC Jun 17,
    // so it overlaps both UTC days.
    const localTue = new Date(2026, 5, 16); // Jun 16 local
    expect(utcDateStringsForLocalDate(localTue)).toEqual([
      '2026-06-16',
      '2026-06-17',
    ]);
  });

  it('returns two UTC dates for the next local day too (the symmetric case)', () => {
    const localWed = new Date(2026, 5, 17); // Jun 17 local
    expect(utcDateStringsForLocalDate(localWed)).toEqual([
      '2026-06-17',
      '2026-06-18',
    ]);
  });

  it('ignores the time-of-day component on the input', () => {
    const morning = new Date(2026, 5, 16, 6, 30, 0); // 6:30 AM ET
    const evening = new Date(2026, 5, 16, 23, 45, 0); // 11:45 PM ET
    expect(utcDateStringsForLocalDate(morning)).toEqual(
      utcDateStringsForLocalDate(evening),
    );
  });

  it('handles month/year rollovers (Dec 31 local in ET)', () => {
    const newYearsEve = new Date(2026, 11, 31); // Dec 31, 2026 ET
    expect(utcDateStringsForLocalDate(newYearsEve)).toEqual([
      '2026-12-31',
      '2027-01-01',
    ]);
  });

  it('handles leap day (Feb 29, 2028)', () => {
    const leap = new Date(2028, 1, 29); // Feb 29, 2028 ET
    expect(utcDateStringsForLocalDate(leap)).toEqual([
      '2028-02-29',
      '2028-03-01',
    ]);
  });

  it('handles DST spring-forward (Mar 8, 2026 ET = UTC-4 starting 02:00 local)', () => {
    const dstDay = new Date(2026, 2, 8); // Mar 8, 2026 ET
    // Local 00:00 ET on a DST-shift day still overlaps two UTC dates because
    // ET is offset from UTC across the full day.
    const result = utcDateStringsForLocalDate(dstDay);
    expect(result).toHaveLength(2);
    expect(result[0]).toBe('2026-03-08');
    expect(result[1]).toBe('2026-03-09');
  });
});

describe('isOnLocalDate (TZ=America/New_York)', () => {
  const localTue = new Date(2026, 5, 16); // Tue Jun 16 ET
  const localWed = new Date(2026, 5, 17); // Wed Jun 17 ET

  it('Argentina vs Algeria @ 01:00 UTC Jun 17 belongs to Tue Jun 16 in ET (the bug)', () => {
    const kickoff = '2026-06-17T01:00:00+00:00';
    expect(isOnLocalDate(kickoff, localTue)).toBe(true);
    expect(isOnLocalDate(kickoff, localWed)).toBe(false);
  });

  it('Austria vs Jordan @ 04:00 UTC Jun 17 belongs to Wed Jun 17 in ET', () => {
    const kickoff = '2026-06-17T04:00:00+00:00';
    expect(isOnLocalDate(kickoff, localTue)).toBe(false);
    expect(isOnLocalDate(kickoff, localWed)).toBe(true);
  });

  it('Portugal vs Congo DR @ 13:00 UTC Jun 17 belongs to Wed Jun 17 in ET', () => {
    const kickoff = '2026-06-17T13:00:00+00:00';
    expect(isOnLocalDate(kickoff, localWed)).toBe(true);
  });

  it('handles ISO strings with explicit Z suffix', () => {
    const kickoff = '2026-06-17T01:00:00Z';
    expect(isOnLocalDate(kickoff, localTue)).toBe(true);
  });

  it('returns false (does not throw) for unparseable input', () => {
    expect(isOnLocalDate('not-a-date', localTue)).toBe(false);
  });

  it('time-of-day on the localDate argument does not change the answer', () => {
    const kickoff = '2026-06-17T01:00:00Z';
    const noonOnTue = new Date(2026, 5, 16, 12, 0, 0);
    expect(isOnLocalDate(kickoff, noonOnTue)).toBe(true);
  });
});
