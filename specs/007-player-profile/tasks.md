# Tasks: 007 – Player Profile Experience

**Input**: Design documents from `/specs/007-player-profile/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Organization**: Tasks are grouped by user story so each story can be implemented and tested independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story (US1–US4)
- Include exact file paths in descriptions

## Path Conventions

- **API**: `services/api/` (Go; sqlc in `sql/`, handlers in `internal/api/`)
- **Mobile**: `apps/mobile/src/` (screens, components, services)

---

## Phase 1: Setup

**Purpose**: Ensure feature branch and docs are ready; no new project init.

- [x] T001 Verify feature branch `007-player-profile` and that specs/007-player-profile/ (spec.md, plan.md, data-model.md, contracts/api.yaml, research.md, quickstart.md) exist

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Database schema and API routing for “my profile” so all user stories can call the same backend.

**Independent Test**: Run migrations; `GET /api/player-profile` returns 404 when no profile; authenticated routes require token.

- [x] T002 Add migration file in services/api/sql/schema/ for 007: player_profiles columns (country_code, club_name, is_free_agent, position enum GK/DEF/MID/FWD, photo_url per data-model.md); add player_profile_traits and player_career_teams tables
- [x] T003 Add sqlc queries in services/api/sql/queries/ (new or extend player_profiles.sql): get/create/update/delete profile by user_id; traits and career_teams CRUD; align with new schema
- [x] T004 Run sqlc generate and goose up; fix compile errors in services/api/internal/database/
- [x] T005 Register authenticated /api/me (or /users/me) router in services/api/internal/api/api.go with RequireAuth; mount GET/POST/PUT/DELETE player-profile routes delegating to new handlers
- [x] T006 Implement GET and POST /api/player-profile in services/api/internal/api/player_profile.go: user from auth context, 404 when no profile; create with age, country_code, club_name, is_free_agent, position; return PlayerProfile DTO (traits/career_teams empty initially)
- [x] T007 Implement PUT and DELETE /api/player-profile in services/api/internal/api/player_profile.go: update fields; delete profile (cascade traits/career_teams); return 404 when no profile
- [x] T008 Add types in apps/mobile/src/types/playerProfile.ts and API client in apps/mobile/src/services/playerProfile.ts: PlayerProfile, PlayerProfileInput, trait codes, CareerTeam; getPlayerProfile, createPlayerProfile, updatePlayerProfile, deletePlayerProfile using makeAuthRequest

**Checkpoint**: Backend supports create/read/update/delete for current user’s profile; mobile can call API with auth.

---

## Phase 3: User Story 1 – Basic Profile Creation & Lifecycle (P1) – MVP

**Goal**: User can create a player profile (age, country, club/free agent, position), edit it, delete it, and see “Create Player Profile” when none exists.

**Independent Test**: Log in, open Player Profile; if no profile see Create screen; submit valid form → profile created and view shows; edit and save; delete with confirm → Create screen again.

- [x] T009 [P] [US1] Add reusable CountryPicker component in apps/mobile/src/components/CountryPicker.tsx: searchable list, ISO 3166-1 alpha-2 codes, optional flags; export selected country code and display name
- [x] T010 [US1] Add Create Player Profile screen in apps/mobile/src/screens/CreatePlayerProfileScreen.tsx: form fields Age (13–60), Country (CountryPicker), Club or Free Agent (text + toggle), Position (GK/DEF/MID/FWD); Next (validate, call createPlayerProfile, navigate to profile); Or Maybe Later (dismiss)
- [x] T011 [US1] Add Player Profile screen in apps/mobile/src/screens/PlayerProfileScreen.tsx: tabs Profile / Stats (placeholder) / Career; Profile tab shows avatar placeholder, age, country, club, position, “Save Profile” for edits; load profile via getPlayerProfile (404 → show Create flow or redirect to CreatePlayerProfileScreen)
- [x] T012 [US1] Wire navigation: add route and entry point (e.g. profile icon or “Player Profile”) in apps/mobile to CreatePlayerProfileScreen when no profile and PlayerProfileScreen when profile exists; after create/delete update nav state
- [x] T013 [US1] Add delete profile flow in apps/mobile: confirm dialog in PlayerProfileScreen (or settings), call deletePlayerProfile, then navigate to Create flow or clear profile state
- [x] T014 [US1] Add edit-profile form or inline edit in PlayerProfileScreen for age, country, club, position; persist via updatePlayerProfile; validation same as create (age 13–60, required country and position)

**Checkpoint**: User can create, view, edit, and delete their single player profile; country picker is reusable.

---

## Phase 4: User Story 2 – Player Traits (P2)

**Goal**: User can select up to 5 traits in a modal and see them as chips on the profile.

**Independent Test**: Open profile → Add Traits → select up to 5 → Save; chips appear; reopen modal, change selection, Save; chips update.

- [x] T015 [P] [US2] Implement PUT /api/player-profile/traits handler in services/api/internal/api/player_profile.go: body { traits: string[] } (max 5); validate trait codes against allowed enum; replace all traits for current user’s profile; return updated traits
- [x] T016 [US2] Add get-traits (or include traits in GET profile) in API and in apps/mobile/src/services/playerProfile.ts: setPlayerProfileTraits(traits); ensure GET profile returns traits array
- [x] T017 [P] [US2] Add PlayerTraitsModal in apps/mobile/src/components/PlayerTraitsModal.tsx: full-screen modal, title “Select Player Traits”, list of 9 traits with icon + name + checkbox, max 5 selected; Save (call API, close); Back/Close dismiss without saving
- [x] T018 [US2] On PlayerProfileScreen Profile tab add “Add Traits” button and trait chips (horizontal wrap); open PlayerTraitsModal on tap; after save refresh profile or local state to show updated traits

**Checkpoint**: Traits persist and display; modal enforces max 5.

---

## Phase 5: User Story 3 – Profile Photo Upload (P3)

**Goal**: User can upload a profile photo from the device; it appears in the avatar and is stored via API.

**Independent Test**: Tap Upload Photo → pick image → loading then avatar shows new photo; invalid file or size shows error and retry.

- [ ] T019 [US3] Implement POST /api/player-profile/photo in services/api/internal/api/player_profile.go: multipart form file; validate MIME (JPEG/PNG) and size (≤ 5 MB); store in S3 (or existing blob store) under player-profiles/; set photo_url on profile; return { photo_url }
- [ ] T020 [US3] Add uploadProfilePhoto in apps/mobile/src/services/playerProfile.ts: pick file (or use image picker), multipart POST to /api/player-profile/photo with auth
- [ ] T021 [US3] In PlayerProfileScreen (and Create flow if needed) add “Upload Photo” CTA; use React Native Image Picker (or expo-image-picker); show loading during upload; on success set avatar to photo_url; on failure show error banner with retry

**Checkpoint**: Photo upload works; avatar updates; errors handled.

---

## Phase 6: User Story 4 – Career Teams (P4)

**Goal**: User can add, edit, and delete career team entries (team name, start year, end year / Present); list ordered by start year descending.

**Independent Test**: Add team (name, 2018, 2020) → appears in list; add second (2020–Present); edit first; delete one; order is descending by start year.

- [ ] T022 [P] [US4] Implement GET/POST/PUT/DELETE /api/player-profile/career-teams in services/api/internal/api/player_profile.go: GET returns list for current user’s profile; POST body { team_name, start_year, end_year? }; PUT/DELETE by career_team_id; validate years (1950–current+1, start ≤ end)
- [ ] T023 [US4] Add listCareerTeams, createCareerTeam, updateCareerTeam, deleteCareerTeam in apps/mobile/src/services/playerProfile.ts
- [ ] T024 [US4] On PlayerProfileScreen add Career section (or Career tab): list entries as “Team Name — start–end”; “Add Career Team” opens form/modal (team name, start year, end year, “Present”); edit/delete per entry; sort by start_year descending

**Checkpoint**: Career teams CRUD works; ordering and validation correct.

---

## Phase 7: Polish & Cross-Cutting

**Purpose**: Documentation, validation, and any cross-cutting fixes.

- [ ] T025 Update quickstart.md in specs/007-player-profile/ with any missing steps (e.g. env for S3, auth token) and verify manual test steps for create, traits, photo, career, delete
- [ ] T026 Run full quickstart flow: backend migrations and API, mobile app, create profile → add traits → upload photo → add career teams → edit → delete; fix any bugs found
- [ ] T027 Ensure API and mobile follow constitution: loading states, error messages, accessibility labels where applicable; no new ESLint/TypeScript errors in apps/mobile for new files

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1**: No dependencies.
- **Phase 2**: Depends on Phase 1. **Blocks** all user stories.
- **Phase 3 (US1)**: Depends on Phase 2. MVP.
- **Phase 4 (US2)**: Depends on Phase 2 (and optionally US1 for UI placement).
- **Phase 5 (US3)**: Depends on Phase 2.
- **Phase 6 (US4)**: Depends on Phase 2.
- **Phase 7**: Depends on Phases 3–6 (or subset to validate).

### User Story Dependencies

- **US1**: After Phase 2. No dependency on US2–US4.
- **US2–US4**: After Phase 2; can be built in parallel after US1 or together.

### Parallel Opportunities

- T009 (CountryPicker) and T015 (traits handler) can run in parallel with other [P] tasks in their phases.
- US2, US3, US4 implementation can be parallelized once Phase 2 and (for nav) US1 screen exist.

---

## Implementation Strategy

### MVP First (User Story 1)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 (US1): Create + View + Edit + Delete profile, country picker.
3. Validate: create profile, edit, delete, see Create again.
4. Deploy/demo MVP.

### Incremental Delivery

1. Phase 2 → foundation.
2. US1 → MVP (create/edit/delete profile).
3. US2 → traits.
4. US3 → photo upload.
5. US4 → career teams.
6. Phase 7 → quickstart and polish.

---

## Notes

- Existing `player_profiles` table and `/player-profiles` routes may differ from 007; Phase 2 migrations and new `/api/player-profile` routes implement the spec without breaking existing behavior.
- Trait codes: LEADERSHIP, FINESSE_SHOT, PLAYMAKER, SPEED_DRIBBLER, LONG_SHOT_TAKER, OUTSIDE_FOOT_SHOT, POWER_HEADER, FLAIR, POWER_FREE_KICK.
- Position enum: GK, DEF, MID, FWD.
- Country: store and send as ISO 3166-1 alpha-2 (e.g. PT for Portugal).
