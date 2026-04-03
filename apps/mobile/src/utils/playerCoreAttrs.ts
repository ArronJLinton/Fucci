import type {PlayerProfile} from '../types/playerProfile';

/**
 * Base core attributes used on the player profile and compare screens.
 * Values are illustrative defaults by position until persisted per-user stats exist.
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
  switch (pos) {
    case 'GK':
      return {speed: 78, shooting: 62, passing: 88, physical: 86, stamina: 84};
    case 'DEF':
      return {speed: 82, shooting: 72, passing: 84, physical: 90, stamina: 86};
    case 'MID':
      return {speed: 88, shooting: 82, passing: 92, physical: 80, stamina: 94};
    case 'FWD':
      return {speed: 96, shooting: 92, passing: 88, physical: 84, stamina: 90};
    default:
      return {speed: 72, shooting: 72, passing: 72, physical: 72, stamina: 72};
  }
}

export function dribblingDefendingForPosition(
  pos: PlayerProfile['position'] | null,
): {dribbling: number; defending: number} {
  switch (pos) {
    case 'GK':
      return {dribbling: 45, defending: 89};
    case 'DEF':
      return {dribbling: 72, defending: 89};
    case 'MID':
      return {dribbling: 90, defending: 72};
    case 'FWD':
      return {dribbling: 92, defending: 45};
    default:
      return {dribbling: 72, defending: 72};
  }
}
