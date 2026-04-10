# Quickstart: Google OAuth Registration & Sign-In

## Prerequisites

- Backend and mobile app run locally.
- Google OAuth client configuration includes:
  - App return URI (iOS/Android): `fucci://auth`
  - Backend callback URI (registered in Google Cloud): `http://localhost:8080/v1/api/auth/google/callback` (or your deployed API callback URL)
- Environment variables configured for Google client credentials and JWT signing.
- Database migration tooling ready for `users` schema updates.

**API base URL:** Routes are served under **`/v1/api`** (e.g. `http://localhost:8080/v1/api`). In this doc, **`{API_BASE}`** means that origin + prefix (no trailing slash), matching mobile `apiConfig.baseURL`.

## 1. Apply database migrations

1. Add columns: `google_id`, `auth_provider`, `avatar_url`, `locale`, `last_login_at`.
2. Backfill existing users: `auth_provider='email'`.
3. Enforce `auth_provider NOT NULL`.
4. Add partial unique index on `google_id` where non-null.

Validation:
- Existing users remain valid with `auth_provider=email`.
- Duplicate non-null `google_id` insert fails as expected.

## 2. Validate backend endpoint contract

Authoritative HTTP contract (including browser OAuth): [`contracts/auth-google.openapi.yaml`](./contracts/auth-google.openapi.yaml). Path suffixes below are relative to **`{API_BASE}`** (see prerequisites).

- `GET {API_BASE}/auth/google/start` — begin server-side OAuth (redirect to Google).
- `GET {API_BASE}/auth/google/callback` — Google `redirect_uri` (redirects into the app with a one-time `code`).
- `POST {API_BASE}/auth/google/exchange` — swap that one-time `code` for `token` + `user` + `is_new`.
- `POST {API_BASE}/auth/google` — native flow with Google `code` + `redirect_uri`.

Endpoint (native): `POST {API_BASE}/auth/google`

Request body:

```json
{
  "code": "google-auth-code",
  "redirect_uri": "fucci://auth"
}
```

Expected results:
- New Google account: `200`, returns `token`, `user`, `is_new=true`.
- Existing Google account: `200`, returns `is_new=false`, updates `last_login_at`.
- `email_verified=false`: `400 EMAIL_NOT_VERIFIED`.
- Invalid/expired code: `400 INVALID_CODE`.
- Invalid token signature: `401 TOKEN_VERIFY_FAILED`.
- Email collision with password account: `409 ACCOUNT_EXISTS_EMAIL`.
- Exchange failure against Google: `500 GOOGLE_API_ERROR`.

## 3. Validate mobile user journeys

The app builds requests from **`apiConfig.baseURL`**, which includes **`/v1/api`**. OAuth browser steps hit **`{API_BASE}/auth/google/...`** as in prerequisites (e.g. `http://localhost:8080/v1/api/auth/google/start?return=...`).

Post-auth navigation is implemented in `resolvePostGoogleAuthRoute` + `dispatchAfterSignInSuccess`:

- **`is_new=true`:** reset to **`CreatePlayerProfile`** (player profile onboarding).
- **`is_new=false`:** reset to **`Main`** with the **Profile** tab selected (signed-in profile surface), unless `returnToDebate` sends the user elsewhere.

### US-01 Sign up with Google

1. Open Sign Up screen and tap **Continue with Google**.
2. Verify the app opens backend-driven OAuth start, e.g. **`GET {API_BASE}/auth/google/start?return=fucci://auth`** (encoded `return` matches `Linking.createURL('auth')` in dev).
3. Complete provider consent.
4. Verify backend callback runs: **`GET {API_BASE}/auth/google/callback`** (Google redirect) exchanges the code and **302** redirects into the app with a one-time handoff `code` and `is_new`; the app then calls **`POST {API_BASE}/auth/google/exchange`** for the session.
5. For a **new** Google account (`is_new=true`), verify navigation to **Create player profile** (`CreatePlayerProfile` screen), not Settings.

### US-02 Log in with existing Google account

1. From the logged-out **Profile** tab (guest auth), tap **Continue with Google** (same entry point pattern as sign-up; there is no separate Login screen).
2. Verify browser flow starts via **`GET {API_BASE}/auth/google/start`** (with the same `return` pattern as US-01).
3. Complete provider consent and callback redirect back into the app (then exchange, as in US-01 step 4).
4. For a **returning** Google user (`is_new=false`), verify navigation to **Main** with the **Profile** tab active (not CreatePlayerProfile). Open **Settings** from Profile if you need to validate account UI there.

### Error and cancellation UX

- User cancellation closes flow without visible error.
- `409 ACCOUNT_EXISTS_EMAIL` shows:
  - "An account with this email already exists. Sign in with your password or link your Google account in Settings."

## 4. Test plan checklist

- Backend unit tests: token parsing/verification logic, account lookup branching, error code mapping.
- Backend integration tests: `POST {API_BASE}/auth/google` happy path + all specified failures.
- Mobile tests: Sign Up / Profile guest auth Google button behavior, cancellation handling, post-auth navigation (`CreatePlayerProfile` when `is_new=true`, Main/Profile when `is_new=false`).
- E2E tests: full US-01 and US-02 across iOS and Android.

## 5. Validation outcomes (2026-04-09)

- Backend integration-focused auth tests:
  - `go test ./internal/api -run TestHandleGoogleAuth -count=1` -> pass
  - Covered new user (`is_new=true`), existing user (`is_new=false`), `EMAIL_NOT_VERIFIED`, `INVALID_CODE`, `TOKEN_VERIFY_FAILED`
- Mobile static validation:
  - `yarn type-check` (apps/mobile) -> pass
  - Sign Up and Profile guest auth Google buttons show in-flight loading state and disable interaction while requests are active.
- Manual flow verification:
  - Backend callback flow (`{API_BASE}/auth/google/start` → `{API_BASE}/auth/google/callback`) completed successfully in local development with `GOOGLE_OAUTH_CALLBACK_URL` configured.
