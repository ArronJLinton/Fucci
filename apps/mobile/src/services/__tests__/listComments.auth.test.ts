import * as api from '../api';
import {listComments} from '../debate';

describe('listComments auth', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('sends Bearer token so current_user_vote can be returned', async () => {
    const makeAuthRequest = jest
      .spyOn(api, 'makeAuthRequest')
      .mockResolvedValue([
        {id: 1, current_user_vote: 'upvote', net_score: 2, reactions: []},
      ]);
    const makeApiRequest = jest.spyOn(api, 'makeApiRequest');

    const list = await listComments(99, 'jwt-token');

    expect(makeAuthRequest).toHaveBeenCalledWith(
      'jwt-token',
      '/debates/99/comments',
      'GET',
    );
    expect(makeApiRequest).not.toHaveBeenCalled();
    expect(list[0]?.current_user_vote).toBe('upvote');
  });

  it('falls back to unauthenticated request when token is absent', async () => {
    const makeApiRequest = jest
      .spyOn(api, 'makeApiRequest')
      .mockResolvedValue([]);
    const makeAuthRequest = jest.spyOn(api, 'makeAuthRequest');

    await listComments(99);

    expect(makeApiRequest).toHaveBeenCalledWith('/debates/99/comments', 'GET');
    expect(makeAuthRequest).not.toHaveBeenCalled();
  });
});
