# Research: Google OAuth Registration & Sign-In

## Decision 1: Single backend endpoint for both register/sign-in

- **Decision**: Implement `POST /auth/google` as one endpoint that branches by user existence and conflict checks.
- **Rationale**: Keeps client integration simple and ensures one policy path for token verification, collision handling, and JWT issuance.
- **Alternatives considered**:
  - Separate `register/google` and `login/google` endpoints (rejected: duplicated logic and inconsistent error handling risk).

## Decision 2: Verify Google identity using auth code exchange + ID token verification

- **Decision**: Accept `{ code, redirect_uri }`, exchange auth code server-side, verify ID token signature/issuer/audience and required claims.
- **Rationale**: Server-side verification prevents forged identity payloads and supports strict error mapping (`INVALID_CODE`, `TOKEN_VERIFY_FAILED`, `GOOGLE_API_ERROR`).
- **Alternatives considered**:
  - Trusting client-provided profile payload without verification (rejected: insecure).
  - Accepting only access token (rejected: lacks direct identity claim guarantees needed for account creation).

## Decision 3: Enforce `email_verified` as hard gate

- **Decision**: Reject verified-signature tokens where `email_verified` is false with `400 EMAIL_NOT_VERIFIED`.
- **Rationale**: Prevents creation/login of unverified identities and aligns with explicit product requirement.
- **Alternatives considered**:
  - Allowing unverified email with reduced permissions (rejected: added complexity and not in scope).

## Decision 4: Account lookup and conflict policy

- **Decision**: Lookup order is `google_id` first, then lowercase email fallback. If fallback finds account with `auth_provider=email`, return `409 ACCOUNT_EXISTS_EMAIL`.
- **Rationale**: `google_id` is stable and unique; email fallback preserves continuity for prior social records while protecting password-based accounts from implicit linking.
- **Alternatives considered**:
  - Auto-link email/password account to Google (rejected: out of scope; explicit account linking is excluded).
  - Lookup by email only (rejected: weaker identity continuity when user email changes).

## Decision 5: User record update policy

- **Decision**: On existing user login, always update `last_login_at`; refresh `avatar_url` if present in token payload; preserve other profile edits not owned by auth flow.
- **Rationale**: Captures login telemetry and keeps profile image reasonably fresh without overwriting user-managed profile fields.
- **Alternatives considered**:
  - Never update avatar (rejected: stale profile risk).
  - Always overwrite all name fields each login (rejected: may clobber user-customized profile names).

## Decision 6: Database migration strategy for safe rollout

- **Decision**:
  1. Add new nullable columns first (`google_id`, `avatar_url`, `locale`, `last_login_at`).
  2. Add `auth_provider` with default `'email'`.
  3. Backfill existing rows to `'email'`.
  4. Enforce `NOT NULL` and create partial unique index on `google_id`.
- **Rationale**: Minimizes migration risk on existing data and avoids downtime due to constraint violations.
- **Alternatives considered**:
  - Add `NOT NULL` without backfill (rejected: would fail on existing rows).

## Decision 7: Mobile auth flow behavior

- **Decision**:
  - Sign Up/Login each get "Continue with Google".
  - iOS uses browser-based auth session with `fucci://auth`.
  - Android uses Google Identity Services with Custom Tab fallback and `com.fucci.app:/oauth2redirect`.
  - On `is_new=true`, navigate to Onboarding Interests; else Home Feed.
  - On user cancel, dismiss silently; on `409 ACCOUNT_EXISTS_EMAIL`, show the specified message.
- **Rationale**: Matches requested UX and keeps onboarding routing deterministic.
- **Alternatives considered**:
  - Show generic error on cancel (rejected: noisy UX).
  - Route all successful auth to one screen (rejected: loses onboarding intent).
