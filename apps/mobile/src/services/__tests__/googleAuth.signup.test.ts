/// <reference types="jest" />

import {resolvePostGoogleAuthRoute} from '../googleAuth';

describe('googleAuth signup routing', () => {
  it('routes new users to onboarding interests screen', () => {
    expect(resolvePostGoogleAuthRoute(true)).toBe('CreatePlayerProfile');
  });

  it('routes existing users to main home flow', () => {
    expect(resolvePostGoogleAuthRoute(false)).toBe('Main');
  });
});

