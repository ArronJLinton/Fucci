/// <reference types="jest" />

import {resolvePostGoogleAuthRoute} from '../googleAuth';

describe('googleAuth login routing', () => {
  it('routes existing users to main home flow', () => {
    expect(resolvePostGoogleAuthRoute(false)).toBe('Main');
  });
});
