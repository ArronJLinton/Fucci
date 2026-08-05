import {makeAuthRequest} from './api';
import type {ReportStoryReason} from './matchStoryApi';

export async function reportDebateComment(
  token: string,
  commentId: number,
  reason: ReportStoryReason = 'inappropriate_content',
): Promise<void> {
  await makeAuthRequest(token, '/reports', 'POST', {
    body: JSON.stringify({
      reportable_type: 'debate_response',
      reportable_id: String(commentId),
      reason,
    }),
  });
}
