## 007 – Player Profile Quickstart

This guide explains how to run and manually test the Player Profile experience.

### 1. Prerequisites

- Node.js + Yarn (matching existing mobile setup).
- Go toolchain (matching `services/api`).
- PostgreSQL running with the Fucci schema migrated.

### 2. Backend – API

1. From repo root:

   ```bash
   cd services/api
   go test ./...
   go run ./cmd/api
   ```

2. Ensure new migrations for `player_profiles`, `player_profile_traits`, and `player_career_teams` are applied (to be added in implementation).

3. Verify profile endpoints (once implemented) using a tool like `curl` or Postman:

   - `GET /api/me/player-profile`
   - `POST /api/me/player-profile`
   - `PUT /api/me/player-profile/traits`
   - `POST /api/me/player-profile/photo`
   - `GET/POST/PUT/DELETE /api/me/player-profile/career-teams`

### 3. Mobile – Player Profile Screen

1. From repo root:

   ```bash
   cd apps/mobile
   yarn install
   yarn start
   ```

2. Open the app in the simulator or on device via Expo.

3. Log in with a test account (see existing debate/auth quickstarts if needed).

4. Navigate to the **Player Profile** entry point once implemented (e.g., profile icon).

5. Validate flows:

   - **Create basic profile**:
     - Fill age, country, club/free agent, and position.
     - Tap **Next** and ensure validation errors appear for missing/invalid fields.
   - **Add traits**:
     - Tap **Add Traits**.
     - Select multiple traits (max 5).
     - Save and confirm chips render on the profile screen and persist on reload.
   - **Upload photo**:
     - Tap **Upload Photo** and select an image.
     - Confirm loading state, success state, and photo rendering.
   - **Career teams**:
     - Add a team with start/end years.
     - Edit and delete entries.
     - Confirm ordering (most recent first).

6. Basic acceptance criteria:

   - Profile data persists across app restarts.
   - Invalid inputs are rejected with clear error messages.
   - Network failures show non-blocking error states and allow retry.

