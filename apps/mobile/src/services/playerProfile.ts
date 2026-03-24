import {ApiRequestError, makeAuthRequest} from './api';
import type {
  PlayerProfile,
  PlayerProfileInput,
  PlayerProfileCareerTeam,
} from '../types/playerProfile';

const BASE = '/me/player-profile';

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
): Promise<PlayerProfile | null> {
  try {
    const data = await makeAuthRequest(token, BASE, 'POST', {
      body: JSON.stringify(body),
    });
    return data as PlayerProfile;
  } catch {
    return null;
  }
}

export async function updatePlayerProfile(
  token: string,
  body: PlayerProfileInput,
): Promise<PlayerProfile | null> {
  try {
    const data = await makeAuthRequest(token, BASE, 'PUT', {
      body: JSON.stringify(body),
    });
    return data as PlayerProfile;
  } catch {
    return null;
  }
}

export async function deletePlayerProfile(token: string): Promise<boolean> {
  try {
    await makeAuthRequest(token, BASE, 'DELETE');
    return true;
  } catch {
    return false;
  }
}

/** PUT /me/player-profile/traits — replace traits (max 5). Returns updated traits. */
export async function setPlayerProfileTraits(
  token: string,
  traits: string[],
): Promise<string[] | null> {
  try {
    const data = await makeAuthRequest(token, `${BASE}/traits`, 'PUT', {
      body: JSON.stringify({traits}),
    });
    return (data as {traits: string[]})?.traits ?? null;
  } catch {
    return null;
  }
}

export type {PlayerProfile, PlayerProfileInput, PlayerProfileCareerTeam};
