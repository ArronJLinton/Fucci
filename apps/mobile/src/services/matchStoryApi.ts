import {makeAuthRequest} from './api';

export type CreateMatchStoryPayload = {
  scope_type?: 'match';
  scope_id: string;
  team_lookup_key: string;
  content_type: 'photo' | 'video';
  media_url: string;
  caption?: string;
};

export type CreateMatchStoryResponse = {
  id: string;
  user_id: number;
  scope_type: string;
  scope_id: string;
  team_lookup_key: string;
  content_type: 'photo' | 'video';
  media_url: string;
  created_at: string;
};

export type ReportStoryReason =
  | 'spam'
  | 'harassment'
  | 'inappropriate_content'
  | 'fake_team'
  | 'other';

export async function createMatchStory(
  token: string,
  payload: CreateMatchStoryPayload,
): Promise<CreateMatchStoryResponse> {
  return makeAuthRequest(token, '/stories', 'POST', {
    body: JSON.stringify({
      scope_type: 'match',
      ...payload,
    }),
  }) as Promise<CreateMatchStoryResponse>;
}

export async function reportMatchStory(
  token: string,
  storyId: string,
  reason: ReportStoryReason = 'inappropriate_content',
): Promise<void> {
  await makeAuthRequest(token, '/reports', 'POST', {
    body: JSON.stringify({
      reportable_type: 'story',
      reportable_id: storyId,
      reason,
    }),
  });
}
