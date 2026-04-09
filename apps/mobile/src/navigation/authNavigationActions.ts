import {CommonActions} from '@react-navigation/native';
import {rootNavigationRef} from './rootNavigation';
import type {ReturnToDebateParams} from '../types/navigation';

const PROFILE_TAB_INDEX = 3;

/** Root stack: only Main, with Profile tab active (logged-out auth surface). */
export function dispatchResetToMainProfileTab() {
  if (!rootNavigationRef.isReady()) {
    return;
  }
  rootNavigationRef.dispatch(
    CommonActions.reset({
      index: 0,
      routes: [
        {
          name: 'Main',
          state: {
            routes: [
              {name: 'Home'},
              {name: 'News'},
              {name: 'Debates'},
              {name: 'Profile', params: {embeddedInTab: true}},
            ],
            index: PROFILE_TAB_INDEX,
          },
        },
      ],
    }),
  );
}

/**
 * Open Profile tab auth UI; optional returnToDebate resumes debate flow after sign-in.
 */
export function rootNavigateToProfileAuth(returnToDebate?: ReturnToDebateParams) {
  if (!rootNavigationRef.isReady()) {
    return;
  }
  rootNavigationRef.navigate('Main', {
    screen: 'Profile',
    params: {
      embeddedInTab: true,
      ...(returnToDebate ? {returnToDebate} : {}),
    },
  });
}

export type AfterSignInOptions = {
  returnToDebate?: ReturnToDebateParams;
  /** Google new user → player profile onboarding. */
  replaceWithCreatePlayerProfile?: boolean;
};

/**
 * After successful sign-in from Profile guest auth or SignUp: go to debate, onboarding, or Profile home.
 */
export function dispatchAfterSignInSuccess(opts: AfterSignInOptions) {
  if (!rootNavigationRef.isReady()) {
    return;
  }

  if (opts.replaceWithCreatePlayerProfile) {
    rootNavigationRef.dispatch(
      CommonActions.reset({
        index: 0,
        routes: [{name: 'CreatePlayerProfile'}],
      }),
    );
    return;
  }

  const rt = opts.returnToDebate;
  if (rt?.match && rt?.debate) {
    rootNavigationRef.dispatch(
      CommonActions.reset({
        index: 0,
        routes: [
          {
            name: 'Main',
            state: {
              routes: [
                {name: 'Home'},
                {name: 'News'},
                {
                  name: 'Debates',
                  state: {
                    routes: [
                      {name: 'MainDebates'},
                      {
                        name: 'SingleDebate',
                        params: {
                          match: rt.match,
                          debate: rt.debate,
                          pendingAction: rt.pendingAction,
                        },
                      },
                    ],
                    index: 1,
                  },
                },
                {name: 'Profile', params: {embeddedInTab: true}},
              ],
              index: 2,
            },
          },
        ],
      }),
    );
    return;
  }

  dispatchResetToMainProfileTab();
}
