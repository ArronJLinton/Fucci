import {COUNTRIES} from '../data/countries';
import type {ComparePlayerSnapshot} from '../types/comparePlayer';
import type {PlayerProfile, PlayerProfileOrDraft} from '../types/playerProfile';
import {displayLevel} from './playerRating';

type AuthUser = {
  display_name?: string | null;
  firstname?: string | null;
  lastname?: string | null;
} | null;

function posAbbrev(
  position: PlayerProfile['position'] | null | undefined,
): string {
  if (position == null) return '';
  switch (position) {
    case 'GK':
      return 'GK';
    case 'DEF':
      return 'CB';
    case 'MID':
      return 'CM';
    case 'FWD':
      return 'ST';
    default:
      return '';
  }
}

/**
 * Build a compare snapshot for the signed-in user's profile. Returns null if the profile
 * is not ready to compare (draft or missing position).
 */
export function buildCompareSnapshotFromProfile(
  profile: PlayerProfileOrDraft,
  user: AuthUser,
  options: {
    isDraftProfile: boolean;
    traitsLen: number;
    completionPercent: number;
  },
): ComparePlayerSnapshot | null {
  if (options.isDraftProfile || profile.position == null) {
    return null;
  }
  const saved = profile as PlayerProfile;
  const displayNameRaw =
    user?.display_name?.trim() ||
    [user?.firstname, user?.lastname].filter(Boolean).join(' ').trim() ||
    'Player';
  const countryCode = profile.country || '';
  const countryLabel =
    COUNTRIES.find(c => c.code === countryCode)?.name?.toUpperCase() ??
    (countryCode ? countryCode.toUpperCase() : '—');
  const team = profile.is_free_agent
    ? 'Free Agent'
    : profile.club?.trim()
      ? profile.club.trim()
      : '—';
  const rating = displayLevel(
    options.traitsLen,
    options.completionPercent,
  );
  return {
    id: `profile-${profile.id}`,
    displayName: displayNameRaw.toUpperCase(),
    age: profile.age,
    countryCode,
    countryLabel,
    team,
    positionAbbrev: posAbbrev(profile.position),
    photoUrl: profile.photo_url,
    rating,
    speed: saved.speed,
    shooting: saved.shooting,
    passing: saved.passing,
    dribbling: saved.dribbling,
    defending: saved.defending,
    physical: saved.physical,
    stamina: saved.stamina,
    valueLabel: '—',
    seasonGoals: 0,
    seasonLabel: '23/24',
  };
}
