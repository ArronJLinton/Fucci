import {
  DebateResponse,
  DebateListItem,
  CardVoteCounts,
} from '../types/debate';
import type {DebateComment} from '../types/debate';
import {apiConfig} from '../config/environment';
import {makeApiRequest, makeAuthRequest} from './api';

/** GET /debates/match?match_id= — fetch existing debates from DB; optional debate_type filter */
export const fetchDebatesByMatch = async (
  matchId: string | number,
  debateType?: 'pre_match' | 'post_match',
): Promise<DebateListItem[]> => {
  try {
    let url = `/debates/match?match_id=${encodeURIComponent(String(matchId))}`;
    if (debateType) {
      url += `&debate_type=${encodeURIComponent(debateType)}`;
    }
    const data = await makeApiRequest(url, 'GET');
    return Array.isArray(data) ? data : [];
  } catch (error) {
    console.error('Error fetching debates by match:', error);
    return [];
  }
};

/** Response shape from POST /debates/generate-set */
export interface GenerateDebateSetResponse {
  debates: DebateResponse[];
  pending?: boolean;
  rateLimited?: boolean;
  partial_set?: boolean;
}

/** POST /debates/generate-set — generate multiple debates for match + type */
export const generateDebateSet = async (
  matchId: string | number,
  debateType: 'pre_match' | 'post_match',
  count: number = 3,
  forceRegenerate?: boolean,
): Promise<GenerateDebateSetResponse | null> => {
  const url = `${apiConfig.baseURL}/debates/generate-set`;
  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: {...apiConfig.headers, 'Content-Type': 'application/json'},
      body: JSON.stringify({
        match_id: String(matchId),
        debate_type: debateType,
        count: count <= 0 ? 3 : Math.min(7, count),
        force_regenerate: !!forceRegenerate,
      }),
    });
    if (response.status === 429) {
      return {debates: [], pending: false, rateLimited: true};
    }
    if (!response.ok) {
      console.error(
        'Error generating debate set:',
        response.status,
        await response.text(),
      );
      return null;
    }
    const data = await response.json();
    if (data?.info && typeof data.info === 'string') {
      return {debates: [], pending: false};
    }
    const debates = Array.isArray(data?.debates) ? data.debates : [];
    return {
      debates,
      pending: !!data?.pending,
      partial_set: !!data?.partial_set,
    };
  } catch (error) {
    console.error('Error generating debate set:', error);
    return null;
  }
};

/** GET /debates/:id — fetch full debate with cards */
export const fetchDebateById = async (
  debateId: number,
): Promise<DebateResponse | null> => {
  try {
    const data = await makeApiRequest(`/debates/${debateId}`, 'GET');
    if (data?.headline && Array.isArray(data?.cards)) {
      return data as DebateResponse;
    }
    return null;
  } catch (error) {
    console.error('Error fetching debate by id:', error);
    return null;
  }
};

/** PUT /debates/:debateId/cards/:cardId/vote — set swipe vote; auth required */
export const setCardVote = async (
  token: string,
  debateId: number,
  cardId: number,
  voteType: 'upvote' | 'downvote',
): Promise<CardVoteCounts | null> => {
  try {
    const data = await makeAuthRequest(
      token,
      `/debates/${debateId}/cards/${cardId}/vote`,
      'PUT',
      {
        body: JSON.stringify({vote_type: voteType}),
        headers: {'Content-Type': 'application/json'},
      },
    );
    return data as CardVoteCounts;
  } catch (error) {
    console.error('Error setting card vote:', error);
    return null;
  }
};

/** GET /debates/:id/comments — list comments with subcomments, net_score, reactions */
export const listComments = async (
  debateId: number,
): Promise<DebateComment[]> => {
  const data = await makeApiRequest(`/debates/${debateId}/comments`, 'GET');
  return Array.isArray(data) ? data : [];
};

/** POST /debates/:debateId/comments — create comment or reply; auth required */
export const createComment = async (
  token: string,
  debateId: number,
  body: {content: string; parent_comment_id?: number | null},
): Promise<DebateComment | null> => {
  try {
    const data = await makeAuthRequest(token, `/debates/${debateId}/comments`, 'POST', {
      body: JSON.stringify({
        content: body.content.trim(),
        ...(body.parent_comment_id != null && {parent_comment_id: body.parent_comment_id}),
      }),
      headers: {'Content-Type': 'application/json'},
    });
    return data as DebateComment;
  } catch (error) {
    console.error('Error creating comment:', error);
    return null;
  }
};

/** PUT /comments/:commentId/vote — set or clear vote; auth required */
export const setCommentVote = async (
  token: string,
  commentId: number,
  voteType: 'upvote' | 'downvote' | null,
): Promise<{net_score: number; vote_type: 'upvote' | 'downvote' | null} | null> => {
  try {
    const data = await makeAuthRequest(token, `/comments/${commentId}/vote`, 'PUT', {
      body: JSON.stringify({vote_type: voteType}),
      headers: {'Content-Type': 'application/json'},
    });
    return data as {net_score: number; vote_type: 'upvote' | 'downvote' | null};
  } catch (error) {
    console.error('Error setting comment vote:', error);
    return null;
  }
};

/** POST /comments/:commentId/reactions — add or toggle emoji; auth required */
export const addCommentReaction = async (
  token: string,
  commentId: number,
  emoji: string,
): Promise<{reactions: Array<{emoji: string; count: number}>} | null> => {
  try {
    const data = await makeAuthRequest(token, `/comments/${commentId}/reactions`, 'POST', {
      body: JSON.stringify({emoji: emoji.trim()}),
      headers: {'Content-Type': 'application/json'},
    });
    return data as {reactions: Array<{emoji: string; count: number}>};
  } catch (error) {
    console.error('Error adding comment reaction:', error);
    return null;
  }
};

/** DELETE /comments/:commentId/reactions?emoji= — remove reaction; auth required */
export const removeCommentReaction = async (
  token: string,
  commentId: number,
  emoji: string,
): Promise<{reactions: Array<{emoji: string; count: number}>} | null> => {
  try {
    const data = await makeAuthRequest(
      token,
      `/comments/${commentId}/reactions?emoji=${encodeURIComponent(emoji.trim())}`,
      'DELETE',
    );
    return data as {reactions: Array<{emoji: string; count: number}>};
  } catch (error) {
    console.error('Error removing comment reaction:', error);
    return null;
  }
};

/** POST /debates/generate — create debate (body: match_id, debate_type) */
export const createDebate = async (
  matchId: string | number,
  debateType: string,
): Promise<DebateResponse | null> => {
  try {
    const data = await makeApiRequest('/debates/generate', 'POST', {
      body: JSON.stringify({
        match_id: String(matchId),
        debate_type: debateType,
      }),
    });
    if (data?.info && typeof data.info === 'string') {
      return null;
    }
    const debate = data?.debate ?? data;
    if (debate?.headline && Array.isArray(debate?.cards)) {
      return debate;
    }
    return null;
  } catch (error) {
    console.error('Error creating debate:', error);
    return null;
  }
};

/** @deprecated Use fetchDebatesByMatch + fetchDebateById or createDebate */
export const fetchDebate = async (
  matchId: number,
  type: string = 'pre_match',
): Promise<DebateResponse | null> => {
  const list = await fetchDebatesByMatch(matchId);
  const existing = list.find(d => d.debate_type === type);
  if (existing) {
    return fetchDebateById(existing.id);
  }
  return createDebate(matchId, type);
};
