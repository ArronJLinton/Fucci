# Quickstart: User Registration and Settings Flow

**Feature**: 005-user-registration-settings  
**Date**: 2026-03-09  
**Target**: Developers running the mobile app and testing sign-up, login, and settings  
**See also**: [spec.md](./spec.md), [plan.md](./plan.md), [contracts/api.yaml](./contracts/api.yaml)

## Prerequisites

- Node 18+, Expo CLI (or EAS)
- Go 1.22+ and PostgreSQL/Redis if running the API locally
- API base URL configured in the mobile app (e.g. `API_URL` or env)

## Environment

**API (services/api)**:

- `JWT_SECRET` – required for auth (e.g. `openssl rand -base64 32`)
- `DB_URL` – PostgreSQL connection string
- Optional: Redis for session/refresh token storage if implemented

**Mobile (apps/mobile)**:

- API base URL must point to your backend. Set in `app.json` → `expo.extra.API_BASE_URL` (e.g. `http://localhost:8080/v1/api`). For Android emulator use `http://10.0.2.2:8080/v1/api`. Use `apps/mobile/scripts/set-env.js` for environment-specific builds if needed.

## Run the API

From repo root:

```bash
cd services/api && go run ./cmd/api
```

Default base URL: `http://localhost:8080` (or as configured). The API is typically mounted at `/v1/api`. Auth routes: `POST /auth/register`, `POST /auth/login`; profile: `GET/PUT /users/profile`; following: `GET /users/me/following` (paths relative to base, e.g. `http://localhost:8080/v1/api`).

## Run the Mobile App

```bash
cd apps/mobile && npx expo start
```

Open the app on a simulator or device. Navigate to Sign Up or Login (e.g. from Profile or onboarding), then after login open **Settings** (from Profile or bottom nav).

## Test Flows

### 1. Sign Up

- Open Sign Up screen.
- Enter: identifier (email or username), password (min 8 chars), first name, last name; optionally add photo.
- Submit; expect 201 and response with user + token (auto sign-in). Request body uses `identifier`, `password`, `first_name`, `last_name`, and optional `photo_url` per [contracts/api.yaml](./contracts/api.yaml).

### 2. Login

- Enter identifier (email or username) and password.
- Optional: toggle “Remember me” (persists session when implemented).
- Submit; expect 200 with user and token.
- “Forgot password?” opens a placeholder screen (full flow in a later phase).

### 3. Settings

- After login, open Settings (e.g. Profile → Settings or tab bar).
- **Following**: List of followed teams/leagues; toggles or manage list (uses `GET /users/me/following`).
- **Player Profile**: View/edit first name, last name, photo (uses `GET/PUT /users/profile`).
- **Team Manager**: Shown for team_manager role; empty state or “Request access” for others.
- **Logout**: Tap to sign out; expect token cleared and navigation to Login or anonymous Home.

## API Contract (005)

Registration request body (aligned with spec):

- `identifier` (required): email or username
- `password` (required): min 8 characters
- `first_name` (required), `last_name` (required)
- `photo_url` (optional)

Profile response includes: `id`, `email`, `username`, `first_name`, `last_name`, `display_name`, `avatar_url`, `role`. See [contracts/api.yaml](./contracts/api.yaml) for full shapes.

## Database

If the existing `users` table does not have `first_name`, `last_name`, or `username`, add a migration (see [data-model.md](./data-model.md)) and run from the repo root:

```bash
yarn migrate
```

This runs the migrator in `services/api`; ensure `DB_URL` is set in the environment or in `services/api/.env`.
