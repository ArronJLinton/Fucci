import fs from 'fs';
import path from 'path';

/**
 * Regression: unauthenticated comment reload + same-vote toggle silently deleted
 * persisted votes while the UI still showed the vote as active.
 */
describe('SingleDebateScreen comment vote regressions', () => {
  const src = fs.readFileSync(
    path.join(__dirname, '../SingleDebateScreen.tsx'),
    'utf8',
  );

  it('loads comments with auth token when available', () => {
    expect(src).toMatch(/listComments\(\s*debateId\s*,\s*token\s*\)/);
  });

  it('applies authoritative server vote_type after setCommentVote', () => {
    expect(src).toContain('res.vote_type');
    expect(src).toMatch(/current_user_vote:\s*appliedVote/);
    expect(src).not.toMatch(/current_user_vote:\s*toSend\s*\?\?/);
  });
});
