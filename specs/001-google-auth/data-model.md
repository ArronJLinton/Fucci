# Data Model: Google OAuth Registration & Sign-In

## Entity: User

Represents application identity for authenticated users.

### Fields

- `id`: UUID, primary key
- `email`: string, required, stored lowercase
- `first_name`: string, optional
- `last_name`: string, optional
- `google_id`: string, nullable, unique when non-null
- `auth_provider`: enum(`email`, `google`, `apple`), required
- `avatar_url`: string, nullable
- `locale`: string, nullable (max 20 chars)
- `last_login_at`: timestamp with timezone, nullable
- `created_at`: timestamp with timezone
- `updated_at`: timestamp with timezone

### Validation Rules

- `email` must be normalized to lowercase before lookup/store.
- `google_id` maps from Google `sub` and must be immutable once set.
- `auth_provider` defaults to `email` for existing/new password users.
- `auth_provider=google` is set for first-time Google registrations.
- If `email_verified=false`, user must not be created or logged in.

### Indexes/Constraints

- Partial unique index: unique `google_id` where non-null.
- Existing uniqueness constraints on `email` remain in effect.

## Entity: GoogleAuthRequest

Represents payload for backend auth exchange.

### Fields

- `code`: string, required, non-empty OAuth authorization code
- `redirect_uri`: string, required, must match configured allowed redirect URIs

## Entity: GoogleClaims (Verified Identity Payload)

Represents verified claims extracted from Google ID token.

### Fields

- `sub`: string, required
- `email`: string, required
- `email_verified`: boolean, required
- `given_name`: string, optional
- `family_name`: string, optional
- `picture`: string, optional
- `locale`: string, optional

## Entity: AuthSessionResponse

Represents successful endpoint response.

### Fields

- `token`: string, signed Fucci JWT
- `user`: object (id, email, first_name, last_name, avatar_url, locale, auth_provider)
- `is_new`: boolean

## State Transitions

1. **New Google User**
   - Input: verified Google claims with unknown `google_id` and unknown email
   - Transition: create `User` with `auth_provider=google`
   - Output: `is_new=true`

2. **Existing Google User**
   - Input: known `google_id`
   - Transition: update `last_login_at` and optional `avatar_url`
   - Output: `is_new=false`

3. **Email/Password Collision**
   - Input: unknown `google_id`, email matches existing `auth_provider=email`
   - Transition: no data mutation
   - Output: `409 ACCOUNT_EXISTS_EMAIL`
