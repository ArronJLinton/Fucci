/// <reference types="jest" />

import {listComments, omitBlockedAuthorComments} from '../debate';
import type {DebateComment} from '../../types/debate';
import * as api from '../api';

function comment(
  partial: Pick<DebateComment, 'id' | 'user_id'> &
    Partial<DebateComment>,
): DebateComment {
  return {
    debate_id: 1,
    content: 'hi',
    user_display_name: 'Fan',
    created_at: '2026-09-24T00:00:00Z',
    net_score: 0,
    reactions: [],
    ...partial,
  };
}

describe('omitBlockedAuthorComments', () => {
  it('removes the blocked author and replies on their threads', () => {
    const comments = [
      comment({
        id: 1,
        user_id: 9,
        subcomments: [comment({id: 2, user_id: 3})],
      }),
      comment({
        id: 4,
        user_id: 3,
        subcomments: [comment({id: 5, user_id: 9})],
      }),
    ];

    expect(omitBlockedAuthorComments(comments, 9)).toEqual([
      comment({
        id: 4,
        user_id: 3,
        subcomments: [],
      }),
    ]);
  });
});

describe('listComments', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('loads anonymously when no session token is available', async () => {
    const makeApiRequest = jest
      .spyOn(api, 'makeApiRequest')
      .mockResolvedValue([]);
    await listComments(12);
    expect(makeApiRequest).toHaveBeenCalledWith('/debates/12/comments', 'GET', {
      headers: undefined,
    });
  });

  it('sends the viewer token so blocked authors are omitted', async () => {
    const makeApiRequest = jest
      .spyOn(api, 'makeApiRequest')
      .mockResolvedValue([]);
    await listComments(12, 'session-token');
    expect(makeApiRequest).toHaveBeenCalledWith('/debates/12/comments', 'GET', {
      headers: {Authorization: 'Bearer session-token'},
    });
  });
});
