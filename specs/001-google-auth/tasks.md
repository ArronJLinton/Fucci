# Tasks: Google OAuth Registration & Sign-In

**Input**: Design documents from `specs/001-google-auth/`  
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/auth-google.openapi.yaml`, `quickstart.md`

**Tests**: Included (TDD required by constitution; tests should be written first and fail before implementation).  
**Organization**: Tasks are grouped by user story for independent implementation and validation.

## Format: `[ID] [P?] [Story] Description`

- `[P]`: Parallelizable task (different files, no unmet dependencies)
- `[Story]`: User story label (`[US1]`, `[US2]`, `[US3]`)
- Every task includes an exact repository-relative file path

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare auth feature scaffolding and config surfaces used by all stories.

- [X] T001 Add Google OAuth env config entries and validation hooks in `services/api/internal/api/api.go`
- [X] T002 [P] Add mobile Google OAuth config constants (redirect URIs, scopes) in `apps/mobile/src/config/googleAuth.ts`
- [X] T003 [P] Add shared backend error-code constants for Google auth responses in `services/api/internal/api/auth_google_errors.go`
- [X] T004 Document required env vars and local setup steps in `services/api/README.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement base data model + reusable backend/mobile plumbing required before user story work.

**⚠️ CRITICAL**: No user story implementation should begin until this phase is complete.

- [X] T005 Create users migration for `google_id`, `auth_provider`, `avatar_url`, `locale`, `last_login_at` in `services/api/sql/schema/20260408170000_add_google_auth_fields_to_users.sql`
- [X] T006 Add partial unique index migration for non-null `google_id` in `services/api/sql/schema/20260408170100_add_users_google_id_partial_unique_index.sql`
- [X] T007 [P] Add sqlc query support for Google lookup/update paths in `services/api/sql/queries/users.sql`
- [X] T008 [P] Regenerate DB query bindings for new users fields in `services/api/internal/db/` (generated sqlc output)
- [X] T009 Implement shared Google code-exchange + ID token verification utility in `services/api/internal/auth/google_oauth.go`
- [X] T010 [P] Implement redirect URI allowlist validator in `services/api/internal/auth/google_oauth.go`
- [X] T011 [P] Add backend unit tests for verifier + redirect URI validation in `services/api/internal/auth/google_oauth_test.go`
- [X] T012 Add typed mobile service method for `POST /auth/google` contract in `apps/mobile/src/services/auth.ts`
- [X] T013 Add shared mobile Google auth launcher utility (iOS/Android) in `apps/mobile/src/services/googleAuth.ts`

**Checkpoint**: Foundation complete; user stories can proceed.

---

## Phase 3: User Story 1 - Register with Google (Priority: P1) 🎯 MVP

**Goal**: A new user can register via Google and be signed in, then routed to onboarding (`is_new=true`).

**Independent Test**: From signed-out state, tap Google on Sign Up, complete consent, verify backend creates user and app navigates to onboarding.

### Tests for User Story 1

- [X] T014 [P] [US1] Add backend integration test for new Google user registration success in `services/api/internal/api/auth_google_test.go`
- [X] T015 [P] [US1] Add backend integration test for `EMAIL_NOT_VERIFIED` rejection in `services/api/internal/api/auth_google_test.go`
- [X] T016 [P] [US1] Add mobile test for Sign Up Google success -> onboarding routing in `apps/mobile/src/services/__tests__/googleAuth.signup.test.ts`

### Implementation for User Story 1

- [X] T017 [US1] Implement `POST /auth/google` handler request parsing + validation in `services/api/internal/api/auth.go`
- [X] T018 [US1] Implement new-user creation flow from verified Google claims in `services/api/internal/api/auth.go`
- [X] T019 [US1] Issue Fucci JWT and return `{ token, user, is_new:true }` in `services/api/internal/api/auth.go`
- [X] T020 [US1] Add "Continue with Google" CTA to sign-up UI in `apps/mobile/src/screens/SignUpScreen.tsx`
- [X] T021 [US1] Wire Sign Up Google success path to onboarding navigation in `apps/mobile/src/screens/SignUpScreen.tsx`
- [X] T022 [US1] Handle Google cancellation silently in sign-up flow in `apps/mobile/src/screens/SignUpScreen.tsx`

**Checkpoint**: US1 independently functional and testable.

---

## Phase 4: User Story 2 - Sign In with Google (Priority: P1)

**Goal**: Existing Google users can sign in and land in home feed (`is_new=false`).

**Independent Test**: Existing Google user signs out, signs back in via Google on Login, receives existing account and home-feed routing.

### Tests for User Story 2

- [ ] T023 [P] [US2] Add backend integration test for existing Google user login (`is_new=false`) in `services/api/internal/api/auth_google_test.go`
- [ ] T024 [P] [US2] Add backend integration test for `TOKEN_VERIFY_FAILED` and `INVALID_CODE` mapping in `services/api/internal/api/auth_google_test.go`
- [ ] T025 [P] [US2] Add mobile test for Login Google success -> home routing in `apps/mobile/src/services/__tests__/googleAuth.login.test.ts`

### Implementation for User Story 2

- [ ] T026 [US2] Implement existing-user lookup by `google_id` then legacy email fallback in `services/api/internal/api/auth.go`
- [ ] T027 [US2] Implement existing-user update policy (`last_login_at`, optional `avatar_url`) in `services/api/internal/api/auth.go`
- [ ] T028 [US2] Return `{ token, user, is_new:false }` for existing Google users in `services/api/internal/api/auth.go`
- [ ] T029 [US2] Add "Continue with Google" CTA to login UI in `apps/mobile/src/screens/LoginScreen.tsx`
- [ ] T030 [US2] Wire Login Google success path to home feed navigation in `apps/mobile/src/screens/LoginScreen.tsx`
- [ ] T031 [US2] Handle Google cancellation silently in login flow in `apps/mobile/src/screens/LoginScreen.tsx`

**Checkpoint**: US2 independently functional and testable.

---

## Phase 5: User Story 3 - Prevent Duplicate Accounts (Priority: P2)

**Goal**: Prevent implicit linking; conflict on password-account email collisions with clear UX.

**Independent Test**: Existing password account with same email attempts Google sign-in and receives `ACCOUNT_EXISTS_EMAIL`; no sign-in or linking occurs.

### Tests for User Story 3

- [ ] T032 [P] [US3] Add backend integration test for `ACCOUNT_EXISTS_EMAIL` conflict in `services/api/internal/api/auth_google_test.go`
- [ ] T033 [P] [US3] Add backend integration test for strict `redirect_uri` mismatch rejection in `services/api/internal/api/auth_google_test.go`
- [ ] T034 [P] [US3] Add mobile UI test for 409 conflict message rendering in `apps/mobile/src/services/__tests__/googleAuth.conflict.test.ts`

### Implementation for User Story 3

- [ ] T035 [US3] Implement `auth_provider=email` collision branch returning 409 error code in `services/api/internal/api/auth.go`
- [ ] T036 [US3] Ensure no account mutation/linking occurs on collision in `services/api/internal/api/auth.go`
- [ ] T037 [US3] Surface exact conflict message in login flow UI in `apps/mobile/src/screens/LoginScreen.tsx`
- [ ] T038 [US3] Surface exact conflict message in sign-up flow UI in `apps/mobile/src/screens/SignUpScreen.tsx`

**Checkpoint**: US3 independently functional and testable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, observability, and documentation updates.

- [ ] T039 [P] Add structured auth event logging for success/failure branches in `services/api/internal/api/auth.go`
- [ ] T040 [P] Add mobile loading/disabled-button state for in-flight Google auth in `apps/mobile/src/screens/LoginScreen.tsx`
- [ ] T041 [P] Add mobile loading/disabled-button state for in-flight Google auth in `apps/mobile/src/screens/SignUpScreen.tsx`
- [ ] T042 Run quickstart validation scenarios and record outcomes in `specs/001-google-auth/quickstart.md`
- [ ] T043 Update contract/examples if response fields changed during implementation in `specs/001-google-auth/contracts/auth-google.openapi.yaml`

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 (Setup): no dependencies
- Phase 2 (Foundational): depends on Phase 1; blocks all user stories
- Phases 3-5 (User Stories): depend on Phase 2 completion
- Phase 6 (Polish): depends on all implemented user stories

### User Story Dependencies

- **US1 (P1)**: starts after Phase 2; no dependency on other stories
- **US2 (P1)**: starts after Phase 2; independent from US1 but reuses same endpoint
- **US3 (P2)**: starts after Phase 2; depends functionally on endpoint branches from US1/US2

### Within Each User Story

- Tests first (fail), then implementation
- Backend branch logic before mobile UX handling
- Route handling after response contract is stable

---

## Parallel Opportunities

- Setup: T002, T003 can run in parallel
- Foundational: T007, T008, T010, T011 can run in parallel after migration plan starts
- US1 tests T014-T016 parallel; UI tasks T020/T021 can split across contributors
- US2 tests T023-T025 parallel; UI tasks T029/T030 parallel
- US3 tests T032-T034 parallel; UI tasks T037/T038 parallel
- Polish: T039-T041 parallel

---

## Parallel Example: User Story 2

```bash
# Parallel test tasks
Task: "T023 [US2] Existing-user backend integration test in services/api/internal/api/auth_google_test.go"
Task: "T024 [US2] Error mapping backend integration test in services/api/internal/api/auth_google_test.go"
Task: "T025 [US2] Mobile login routing test in apps/mobile/src/services/__tests__/googleAuth.login.test.ts"

# Parallel UI tasks
Task: "T029 [US2] Add Google CTA in apps/mobile/src/screens/LoginScreen.tsx"
Task: "T030 [US2] Wire home routing in apps/mobile/src/screens/LoginScreen.tsx"
```

---

## Implementation Strategy

### MVP First (US1)

1. Complete Phase 1 and Phase 2
2. Deliver Phase 3 (US1) end-to-end
3. Validate US1 independently before expanding scope

### Incremental Delivery

1. Add US1 (registration)
2. Add US2 (existing-user login)
3. Add US3 (collision/conflict safeguards)
4. Finish polish and full regression validation

### Suggested MVP Scope

- Phases 1-3 only (through T022) for first releasable increment.
