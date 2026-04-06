import {ApiRequestError, makeAuthRequest} from './api';
import type {
  PlayerProfile,
  PlayerProfileInput,
  PlayerProfileCareerTeam,
} from '../types/playerProfile';
import type {ComparePlayerSnapshot} from '../types/comparePlayer';
import {COUNTRIES} from '../data/countries';

const BASE = '/player-profile';

function isNotFoundError(error: unknown): boolean {
  if (error instanceof ApiRequestError) return error.status === 404;
  if (error instanceof Error && /\b404\b/.test(error.message)) return true;
  return false;
}

export async function getPlayerProfile(
  token: string,
): Promise<PlayerProfile | null> {
  try {
    const data = await makeAuthRequest(token, BASE, 'GET');
    return data as PlayerProfile;
  } catch (error) {
    if (isNotFoundError(error)) return null;
    throw error;
  }
}

export async function createPlayerProfile(
  token: string,
  body: PlayerProfileInput,
): Promise<PlayerProfile> {
  const data = await makeAuthRequest(token, BASE, 'POST', {
    body: JSON.stringify(body),
  });
  return data as PlayerProfile;
}

export async function updatePlayerProfile(
  token: string,
  body: PlayerProfileInput,
): Promise<PlayerProfile> {
  const data = await makeAuthRequest(token, BASE, 'PUT', {
    body: JSON.stringify(body),
  });
  return data as PlayerProfile;
}

export async function deletePlayerProfile(token: string): Promise<void> {
  await makeAuthRequest(token, BASE, 'DELETE');
}

/** PUT /player-profile/traits — replace traits. Returns updated traits. */
export async function setPlayerProfileTraits(
  token: string,
  traits: string[],
): Promise<string[]> {
  const data = await makeAuthRequest(token, `${BASE}/traits`, 'PUT', {
    body: JSON.stringify({traits}),
  });
  const list = (data as {traits?: unknown})?.traits;
  if (!Array.isArray(list) || !list.every(t => typeof t === 'string')) {
    throw new ApiRequestError('Invalid traits response', 500);
  }
  return list;
}

/** GET /player-profile/catalog — compare-opponent candidates from persisted player profiles. */
export async function listComparePlayerCatalog(
  token: string,
  query = '',
): Promise<ComparePlayerSnapshot[]> {
  const q = query.trim();
  const path = q ? `${BASE}/catalog?q=${encodeURIComponent(q)}` : `${BASE}/catalog`;
  const data = await makeAuthRequest(token, path, 'GET');
  const players = (data as {players?: unknown})?.players;
  if (!Array.isArray(players)) {
    throw new ApiRequestError('Invalid compare player catalog response', 500);
  }
  return (players as ComparePlayerSnapshot[]).map(p => {
    const countryCode = (p.countryCode || '').toUpperCase();
    const countryLabel =
      COUNTRIES.find(c => c.code === countryCode)?.name?.toUpperCase() ??
      (countryCode || '—');
    return {
      ...p,
      countryCode,
      countryLabel,
    };
  });
}

export type {PlayerProfile, PlayerProfileInput, PlayerProfileCareerTeam};
