import fs from 'fs';
import path from 'path';
import type {RootStackParamList} from '../../types/navigation';

/**
 * Regression locks for MatchTeamShortsScreen critical navigation / pager bugs.
 * Full screen render tests are heavy; these catch the exact PR #79 regressions.
 */
describe('MatchTeamShortsScreen regressions', () => {
  const src = fs.readFileSync(
    path.join(__dirname, '../MatchTeamShortsScreen.tsx'),
    'utf8',
  );

  it('does not navigate to removed Login stack for add-story auth', () => {
    type HasLogin = 'Login' extends keyof RootStackParamList ? true : false;
    const hasLogin: HasLogin = false;
    expect(hasLogin).toBe(false);

    expect(src).toContain('rootNavigateToProfileAuth');
    expect(src).not.toMatch(/navigate\(\s*['"]Login['"]/);
  });

  it('resets page state when pagerIdentityKey changes (report/delete remount)', () => {
    expect(src).toContain('pagerIdentityKey');
    expect(src).toMatch(
      /useEffect\(\s*\(\)\s*=>\s*\{[\s\S]*?pageRef\.current\s*=\s*0[\s\S]*?\},\s*\[pagerIdentityKey/,
    );
  });
});
