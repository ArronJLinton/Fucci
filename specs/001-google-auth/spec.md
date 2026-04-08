# Feature Specification: Google Auth Registration Sign-In

**Feature Branch**: `001-google-auth`  
**Created**: 2026-04-08  
**Status**: Draft  
**Input**: User description: "Implement google auth for registration/sign-in."

## Clarifications

### Session 2026-04-08

- Q: If Google email matches an existing password account, should we auto-link or block? → A: Return `ACCOUNT_EXISTS_EMAIL` conflict and do not sign in.
- Q: What is the identity lookup order for existing users? → A: Use `google_id` first, then email fallback only for legacy records without `google_id`.
- Q: How should unverified Google emails be handled? → A: Reject both registration and sign-in with `EMAIL_NOT_VERIFIED`.
- Q: How should `redirect_uri` be validated? → A: Require exact match against a configured allowlist and reject mismatches.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Register with Google (Priority: P1)

A new user chooses Google during registration and successfully creates an account without manually entering credentials.

**Why this priority**: Registration is the highest-friction entry point; reducing signup effort directly improves conversion.

**Independent Test**: Can be fully tested by starting from a signed-out state, choosing Google on registration, completing consent, and confirming the user lands in an authenticated session with a new account.

**Acceptance Scenarios**:

1. **Given** a signed-out person without an existing account, **When** they select "Continue with Google" during registration and complete Google consent, **Then** a new account is created and they are signed in.
2. **Given** a signed-out person who cancels Google consent, **When** they return to the app, **Then** no account is created and they remain signed out with a clear cancellation message.

---

### User Story 2 - Sign In with Google (Priority: P1)

A returning user uses Google sign-in and accesses their existing account.

**Why this priority**: Fast and reliable access for returning users is core authentication value and required for daily use.

**Independent Test**: Can be tested by creating a user tied to a Google identity, signing out, then signing in with the same Google account and confirming account continuity.

**Acceptance Scenarios**:

1. **Given** a user with an existing account linked to Google, **When** they choose "Continue with Google" on sign-in and complete consent, **Then** they are signed into the same existing account.
2. **Given** a signed-out user who starts Google sign-in and the provider is temporarily unavailable, **When** the attempt fails, **Then** the user remains signed out and sees an actionable retry message.

---

### User Story 3 - Prevent Duplicate Accounts (Priority: P2)

A user who already registered with email/password can still use Google without ending up with duplicate profiles.

**Why this priority**: Duplicate account prevention protects user data consistency and avoids support burden.

**Independent Test**: Can be tested by signing up with one method first, then attempting Google sign-in with the same verified email, and confirming only one account remains accessible.

**Acceptance Scenarios**:

1. **Given** a user account already exists with the same verified email and that account uses password authentication, **When** the user signs in with Google, **Then** the system denies sign-in with an account-exists conflict and does not create or link accounts.

---

### Edge Cases
- User denies Google consent after initiating the flow.
- Google returns an email that is unverified or unavailable.
- The same Google account is already linked to another internal account.
- Session state expires between starting and finishing Google auth.
- Repeated rapid taps on Google auth do not create multiple in-flight auth attempts.

## Requirements *(mandatory)*

### Functional Requirements
- **FR-001**: The system MUST present a "Continue with Google" option on both registration and sign-in entry points.
- **FR-002**: The system MUST allow a new user to complete registration using a valid Google identity without requiring a password during that flow.
- **FR-003**: The system MUST allow a returning user to sign in using Google and land in their existing account.
- **FR-004**: The system MUST prevent duplicate account creation when a Google identity corresponds to an existing user record.
- **FR-010**: The system MUST look up users by Google subject identifier (`google_id`) as the primary key and only use email fallback for legacy records that do not yet have a stored `google_id`.
- **FR-009**: If Google email matches an existing account whose authentication method is password-based, the system MUST return an account-exists conflict and MUST NOT auto-link or sign in the user.
- **FR-005**: The system MUST clearly communicate failed or cancelled Google auth attempts and keep the user signed out when authentication is incomplete.
- **FR-006**: The system MUST establish an authenticated session immediately after successful Google registration or sign-in.
- **FR-007**: The system MUST record auditable authentication events for successful and failed Google auth attempts.
- **FR-008**: The system MUST allow users who originally registered with Google to continue accessing their account via Google sign-in in later sessions.
- **FR-011**: The system MUST reject any Google auth attempt where `email_verified` is false and return `EMAIL_NOT_VERIFIED`, regardless of whether the user is new or existing.
- **FR-012**: The system MUST validate `redirect_uri` as an exact match against configured allowed redirect URIs and reject mismatches.

### Key Entities *(include if feature involves data)*

- **User Account**: Represents a person using the app; includes account status, primary email, and profile identity.
- **External Identity Link**: Represents the association between a user account and a Google identity provider record; includes provider type, provider subject identifier, and link state.
- **Authentication Session**: Represents a signed-in state after successful auth; includes session start, expiration, and user association.
- **Authentication Event**: Represents a traceable record of auth outcomes; includes event type (success/failure/cancelled), timestamp, and related user/account context.

### Assumptions

- Existing registration and sign-in experiences remain available; Google is an additional auth option, not a replacement.
- Google subject identifier (`google_id`) is the canonical identity key for lookup; email is a fallback only for legacy records without `google_id`.
- Standard consent and account-selection UX is handled through Google-provided flow before control returns to the app.
- Allowed redirect URIs are preconfigured per platform and enforced with exact matching at request validation time.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: At least 90% of users who start Google registration complete account creation successfully on first attempt.
- **SC-002**: At least 95% of successful Google sign-in attempts result in users reaching an authenticated state within 10 seconds.
- **SC-003**: Duplicate-account incidents related to Google auth remain below 1% of monthly Google-authenticated users.
- **SC-004**: Support requests related to login friction decrease by at least 25% within one release cycle after launch.
