# Tasks: User Registration and Settings Flow

**Input**: Design documents from `specs/005-user-registration-settings/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Organization**: Tasks are grouped by user story so each story can be implemented and tested independently. Backend auth and user APIs already exist (001); this feature extends them and adds mobile UI.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story (US1 Sign Up, US2 Login, US3 Settings)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify project structure and environment for auth and settings work.

- [x] T001 Verify apps/mobile and services/api structure; ensure Expo and Go 1.22+ and DB/Redis per quickstart in specs/005-user-registration-settings/quickstart.md
- [x] T002 [P] Ensure JWT_SECRET and API base URL are documented for mobile (e.g. apps/mobile/.env.example or README) per specs/005-user-registration-settings/quickstart.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Backend and schema support for registration (with auto sign-in), login by identifier, and profile. Must complete before any user story UI.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T003 Add DB migration in services/api/sql/migrations/ for users table: add username VARCHAR(100) UNIQUE nullable if not present; ensure firstname, lastname, avatar_url exist (align with specs/005-user-registration-settings/data-model.md)
- [x] T004 Extend POST /auth/register in services/api/internal/api/users.go to accept optional avatar_url/photo_url; return user + token in response (201) so client can auto sign-in per spec clarification
- [x] T005 Extend handleCreateUser in services/api/internal/api/users.go to return full UserResponse and JWT token (reuse auth.GenerateToken) on 201 for auto sign-in
- [x] T006 Extend login to accept identifier (email or username) in services/api/internal/api/auth.go: add identifier field to LoginRequest; resolve user by email or username (add GetUserByUsername or equivalent in services/api/internal/database if needed)
- [x] T007 Enforce password rules in services/api/internal/auth (min length 8, at least one letter and one number) and document in contracts; ensure ValidatePasswordStrength matches specs/005-user-registration-settings/spec.md FR-006
- [x] T008 [P] Add GET /users/me/following (or equivalent) in services/api for Following tab if not present; return list of followed leagues and teams per specs/005-user-registration-settings/contracts/api.yaml

**Checkpoint**: Backend supports register (with token response), login by identifier, profile GET/PUT, password rules, and following list. Mobile can proceed.

---

## Phase 3: User Story 1 – Sign Up (Priority: P1) – MVP

**Goal**: New user can create an account with identifier (email/username), password, first name, last name, optional photo; on success they are automatically signed in and taken to the main app.

**Independent Test**: Complete sign-up form, submit; user is signed in and sees Home or Profile without manually logging in.

### Implementation for User Story 1

- [x] T009 [P] [US1] Add auth API helpers in apps/mobile/src/services/api.ts (or auth.ts): register(body: { identifier, password, first_name, last_name, photo_url? }) calling POST /auth/register; map to backend firstname/lastname/email and optional avatar_url
- [x] T010 [US1] Create SignUpScreen in apps/mobile/src/screens/SignUpScreen.tsx with fields: identifier (email or username), password, first name, last name, optional photo picker/placeholder; client-side validation (required fields, password min 8 chars + one letter + one number)
- [x] T011 [US1] On SignUpScreen submit in apps/mobile/src/screens/SignUpScreen.tsx: call register API; on 201 store token (and user) in secure storage or auth context; navigate to main app (e.g. Home or Profile); show inline validation errors on 400
- [x] T012 [US1] Add "Already have an account? Login" link on SignUpScreen in apps/mobile/src/screens/SignUpScreen.tsx that navigates to Login screen
- [x] T013 [US1] Register SignUpScreen in app navigation and expose entry point (e.g. from Profile when unauthenticated or onboarding) in apps/mobile/src/navigation or App entry

**Checkpoint**: User can sign up and is auto signed in; can navigate to Login from Sign Up.

---

## Phase 4: User Story 2 – Login (Priority: P1)

**Goal**: Returning user can sign in with identifier (email/username) and password; optional "Remember me" and "Forgot password?" link to placeholder screen.

**Independent Test**: Enter valid credentials, tap Login; reach main app. Tap "Forgot password?" and see placeholder/Coming soon.

### Implementation for User Story 2

- [ ] T014 [P] [US2] Add login API helper in apps/mobile/src/services/api.ts: login(body: { identifier, password }) calling POST /auth/login; return token and user; optional persist refresh_token or long-lived token when "Remember me" (e.g. Expo SecureStore)
- [ ] T015 [US2] Create LoginScreen in apps/mobile/src/screens/LoginScreen.tsx with fields: identifier, password; "Remember me" checkbox; "Login" button; "Forgot password?" link; "Don't have an account? Sign Up" link
- [ ] T016 [US2] On LoginScreen submit in apps/mobile/src/screens/LoginScreen.tsx: call login API; on 200 store token (and optionally user); if "Remember me" use persistent storage (e.g. SecureStore), else session-only; navigate to main app; show error message on 401
- [ ] T017 [US2] Create ForgotPasswordPlaceholderScreen in apps/mobile/src/screens/ForgotPasswordPlaceholderScreen.tsx showing "Coming soon" or placeholder message; "Forgot password?" on LoginScreen navigates to it
- [ ] T018 [US2] Register LoginScreen and ForgotPasswordPlaceholderScreen in navigation; ensure SignUpScreen "Already have an account? Login" and Profile (when logged out) open LoginScreen

**Checkpoint**: User can log in and reach main app; Forgot password shows placeholder.

---

## Phase 5: User Story 3 – Settings and Profile (Priority: P2)

**Goal**: Authenticated user can open Settings, see profile summary, switch tabs (Following, Player Profile, Team Manager), and log out with confirmation.

**Independent Test**: Log in, open Settings from Profile/bottom nav; switch tabs; tap Logout and confirm; user is signed out and returned to Login or Home.

### Implementation for User Story 3

- [ ] T019 [US3] Replace stub in apps/mobile/src/screens/SettingsScreen.tsx with full layout: header with back; profile summary (avatar, name, email) with chevron linking to edit; tabs: Following | Player Profile | Team Manager; Logout action at bottom
- [ ] T020 [US3] Implement Following tab in apps/mobile/src/screens/SettingsScreen.tsx: fetch GET /users/me/following; display list of leagues and teams with toggles to follow/unfollow; empty state when none
- [ ] T021 [US3] Implement Player Profile tab in apps/mobile/src/screens/SettingsScreen.tsx: load GET /users/me; editable fields first name, last name, photo (avatar); save via PUT /users/me with validation and error display
- [ ] T022 [US3] Implement Team Manager tab in apps/mobile/src/screens/SettingsScreen.tsx: if user role is team_manager show team management content or link; else show empty state or "Request access" message
- [ ] T023 [US3] Implement Logout in apps/mobile/src/screens/SettingsScreen.tsx: on Logout tap show confirmation dialog ("Log out?" with Cancel / Log out); on confirm clear token and user state and navigate to Login or anonymous Home
- [ ] T024 [US3] Ensure Settings is reachable from Profile or bottom nav when logged in; style per FUCCI dark theme and blue accents per spec UI flow summary

**Checkpoint**: Settings with all three tabs and Logout (with confirmation) work; profile editable in Player Profile tab.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Consistency, validation, and docs.

- [ ] T025 [P] Add client-side password validation (min 8 chars, at least one letter and one number) and display rules on SignUpScreen and optionally LoginScreen per specs/005-user-registration-settings/spec.md FR-006
- [ ] T026 Ensure loading states and error messages for all auth and settings API calls in apps/mobile (SignUpScreen, LoginScreen, SettingsScreen)
- [ ] T027 Run through specs/005-user-registration-settings/quickstart.md: sign-up, login, settings tabs, logout; fix any contract or env gaps

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies – start immediately.
- **Phase 2 (Foundational)**: Depends on Phase 1 – **blocks** all user stories.
- **Phase 3 (US1 Sign Up)**: Depends on Phase 2 – can start when backend register returns token and accepts new fields.
- **Phase 4 (US2 Login)**: Depends on Phase 2 – can run in parallel with Phase 3 after Phase 2.
- **Phase 5 (US3 Settings)**: Depends on Phase 2; uses profile and following APIs – can run after or in parallel with US1/US2.
- **Phase 6 (Polish)**: Depends on completion of desired user stories.

### User Story Dependencies

- **US1 (Sign Up)**: No dependency on US2/US3; needs T003–T005 (register + token).
- **US2 (Login)**: No dependency on US1/US3; needs T006 (login by identifier).
- **US3 (Settings)**: No dependency on US1/US2 for core flow; needs T008 (following) and existing GET/PUT /users/me.

### Parallel Opportunities

- T002 can run in parallel with T001.
- T008 can run in parallel with T003–T007 (different handlers).
- After Phase 2, US1 (T009–T013) and US2 (T014–T018) can be implemented in parallel.
- US3 (T019–T024) can start once profile/following APIs are available; can overlap with US1/US2.
- T025, T027 are [P] or standalone.

---

## Parallel Example: User Story 1

```text
# After Phase 2:
T009 (auth API register) → then T010 (SignUpScreen UI) → T011 (submit + token + nav) → T012 (Login link) → T013 (navigation)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 (Setup).
2. Complete Phase 2 (Foundational) – register returns token, login by identifier, password rules.
3. Complete Phase 3 (US1 Sign Up) – SignUpScreen, auto sign-in, navigation.
4. **STOP and VALIDATE**: Sign up a user and confirm they land in the app without manual login.
5. Deploy or demo if ready.

### Incremental Delivery

1. Phase 1 + 2 → backend and schema ready.
2. Add US1 (Sign Up) → test independently → MVP.
3. Add US2 (Login) → test independently.
4. Add US3 (Settings) → test independently.
5. Phase 6 (Polish) last.

### Parallel Team Strategy

- One developer: Phase 1 → 2 → 3 → 4 → 5 → 6 in order.
- Two developers: After Phase 2, Dev A: US1 (T009–T013), Dev B: US2 (T014–T018); then Dev B or A: US3 (T019–T024).

---

## Notes

- Backend uses `firstname`/`lastname` in code and DB; spec and contract use `first_name`/`last_name` – map in API layer or client.
- [P] = parallelizable (different files or no ordering requirement).
- [US1]/[US2]/[US3] map to Sign Up, Login, Settings for traceability.
- Commit after each task or logical group; validate at checkpoints.
