---
description: "Task list for 008 photo upload (Cloudinary)"
---

# Tasks: Photo upload (Cloudinary)

**Input**: Design documents from `/Users/arronlinton/Desktop/lab/FucciShop/Fucci/specs/008-photo-upload-cloudinary/`  
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [data-model.md](./data-model.md), [contracts/api.yaml](./contracts/api.yaml), [research.md](./research.md), [quickstart.md](./quickstart.md)

**Tests**: Included per **FR-007** and constitution (signature/URL validation + persistence).

**Organization**: Tasks are grouped by user story for independent increments.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no blocking deps within phase)
- **[Story]**: [US1], [US2], [US3] for user-story phases only

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Env wiring, mobile permissions, dependencies.

- [X] T001 Add `CLOUDINARY_CLOUD_NAME`, `CLOUDINARY_API_KEY`, `CLOUDINARY_API_SECRET` (and optional preset notes) to `services/api/.env.example` with short comments
- [X] T002 [P] Update `specs/008-photo-upload-cloudinary/quickstart.md` to reference `services/api/.env.example` and local API URL for signature smoke tests
- [X] T003 [P] Add iOS `NSCameraUsageDescription` and `NSPhotoLibraryUsageDescription` (and Android `CAMERA` / `READ_MEDIA_IMAGES` or legacy storage as required by Expo SDK) in `apps/mobile/app.json` for `expo-image-picker`
- [X] T004 Add dependency `expo-image-picker` in `apps/mobile/package.json` and run install from `apps/mobile/` (lockfile updated)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Signed upload + shared URL validation + mobile upload helper. **No user-story UI work** should ship before core signing/validation exists.

**⚠️ CRITICAL**: User story phases depend on this phase.

- [X] T005 Implement Cloudinary upload signing and `max_upload_bytes` (5242880) response field in `services/api/internal/api/cloudinary.go` per `specs/008-photo-upload-cloudinary/contracts/api.yaml`
- [X] T006 [P] Implement HTTPS URL allowlist helper (host + folder prefix by `context`) in `services/api/internal/api/cloudinary.go` for reuse by profile handlers
- [X] T007 Register authenticated `POST /upload/cloudinary/signature` in `services/api/internal/api/api.go` and plumb `CLOUDINARY_*` from `services/api/main.go` (or config) into `api.Config`
- [X] T008 [P] Add unit tests for signing + URL validation in `services/api/internal/api/cloudinary_test.go` (FR-007)
- [X] T009 Implement `requestSignature`, size check (≤ 5 MB), and multipart upload to Cloudinary returning `secure_url` in `apps/mobile/src/services/cloudinaryUpload.ts` (uses `makeAuthRequest` + `fetch` to Cloudinary)

**Checkpoint**: Signature endpoint + mobile helper callable; foundation ready for US1/US2.

---

## Phase 3: User Story 1 — Account profile photo (Priority: P1) — MVP

**Goal**: User sets **account avatar** from **library or camera**; `avatar_url` stored as Cloudinary HTTPS URL; errors are visible; **FR-009** omit field / no null-clear.

**Independent Test**: After upload, `GET` user profile shows new `avatar_url`; invalid URL or oversize rejected; Settings shows image.

### Implementation for User Story 1

- [X] T010 [US1] Extend `PUT` profile handler in `services/api/internal/api/auth.go` to validate `avatar_url` with `services/api/internal/api/cloudinary.go` helper when present; reject invalid URLs with 400 (FR-005, FR-009)
- [X] T011 [US1] Add HTTP tests for profile `avatar_url` update (valid Cloudinary URL vs bad host) in `services/api/internal/api/auth_test.go` (create file if missing) or extend existing API test harness (FR-007)
- [X] T012 [P] [US1] Ensure `updateProfile` in `apps/mobile/src/services/auth.ts` sends `avatar_url` after upload and handles `ApiRequestError` per existing patterns
- [X] T013 [US1] Implement avatar flow: ImagePicker (library + camera), permission-denied copy, loading state, call `apps/mobile/src/services/cloudinaryUpload.ts` with `context: 'avatar'`, then persist via `apps/mobile/src/screens/SettingsScreen.tsx`
- [X] T014 [P] [US1] Add Jest tests for `apps/mobile/src/services/cloudinaryUpload.ts` with mocked `fetch` / auth (FR-007)

**Checkpoint**: User Story 1 complete and testable on device.

---

## Phase 4: User Story 2 — Player profile photo (Priority: P1)

**Goal**: User sets **player profile photo** only from **Player Profile** after profile exists; **not** on `CreatePlayerProfileScreen` (**FR-008**); `photo_url` persisted.

**Independent Test**: `GET /player-profile` returns `photo_url` after upload; hero shows image; Create flow still has no photo upload.

### Implementation for User Story 2

- [ ] T015 [US2] Add optional `PhotoURL *string` `json:"photo_url"` to `PlayerProfileInput` in `services/api/internal/api/player_profile.go`; on `PUT`/`POST`, when set, validate with Cloudinary helper and pass into `database.UpdatePlayerProfileRowParams` / upsert (replace `PhotoUrl: profile.PhotoUrl` preserve-only logic); omit means unchanged (**FR-009**)
- [ ] T016 [US2] Extend `services/api/internal/api/player_profile_test.go` for `photo_url` on PUT (valid URL + invalid URL cases) (FR-007)
- [ ] T017 [P] [US2] Add optional `photo_url` to `PlayerProfileInput` in `apps/mobile/src/types/playerProfile.ts` and thread through `apps/mobile/src/services/playerProfile.ts` for `createPlayerProfile` / `updatePlayerProfile`
- [ ] T018 [US2] Wire player hero photo upload in `apps/mobile/src/screens/PlayerProfileScreen.tsx` using `apps/mobile/src/services/cloudinaryUpload.ts` with `context: 'player_profile'`; **verify** `apps/mobile/src/screens/CreatePlayerProfileScreen.tsx` has **no** photo upload UI (FR-008)

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 — Replace photos (Priority: P2)

**Goal**: Second upload **replaces** stored URL; **no** explicit remove/clear (**FR-009**).

**Independent Test**: Upload twice; latest `secure_url` wins in DB; no `null` clear path in API.

### Implementation for User Story 3

- [ ] T019 [US3] Add regression tests in `services/api/internal/api/auth_test.go` and `services/api/internal/api/player_profile_test.go` that two sequential valid URL updates change stored values (FR-007)
- [ ] T020 [US3] Verify UI: no “remove” / “clear photo” control in `apps/mobile/src/screens/SettingsScreen.tsx` and `apps/mobile/src/screens/PlayerProfileScreen.tsx`; placeholders only when URL never set (FR-009)

**Checkpoint**: Replace-only behavior covered by tests + UI review.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T021 [P] Add short comment in `specs/007-player-profile/contracts/api.yaml` or `services/api/internal/api/player_profile.go` if legacy `POST /player-profile/photo` multipart is deprecated in favor of 008 signed upload + JSON URL (avoid duplicate implementations)
- [ ] T022 [P] Refresh root or `services/api/README.md` only if needed to mention Cloudinary env vars (keep minimal; prefer `quickstart.md`)
- [ ] T023 Execute manual validation steps in `specs/008-photo-upload-cloudinary/quickstart.md` on iOS/Android

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1** → **Phase 2** → **Phases 3–5** (US1 can start after Phase 2; US2 needs Phase 2 + preferably `PlayerProfileInput` work order after shared helper)
- **Phase 6** after desired stories complete

### User Story Dependencies

- **US1**: After Phase 2. No dependency on US2.
- **US2**: After Phase 2. Uses same `cloudinaryUpload.ts` as US1 (completed in Phase 2/T009 + US1 may harden helper).
- **US3**: After US1 and US2 behaviors exist (replace is same code paths).

### Parallel Opportunities

- **Phase 1**: T002, T003 in parallel with T001; T004 after or parallel if different owner
- **Phase 2**: T006, T008 parallel after T005 started; T009 mobile can parallel T007 once signature contract stable
- **US1**: T012, T014 parallel after T010–T011
- **US2**: T017 parallel to T015–T016 once types known
- **Phase 6**: T021, T022 parallel

### Parallel Example: Phase 2

```bash
# After T005 started: split validation helper (T006) and tests (T008)
# Mobile T009 can proceed in parallel with T007 once POST body/JSON for signature is fixed
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Complete Phase 1 and Phase 2  
2. Complete Phase 3 (US1) — account avatar end-to-end  
3. **STOP**: Validate independent test for US1 on a device  

### Incremental Delivery

1. Add Phase 4 (US2) — player profile photo  
2. Add Phase 5 (US3) — replace regression + UI verification  
3. Phase 6 — docs and smoke  

### Suggested MVP Scope

**US1 (account avatar)** only: Phases 1–3. Delivers visible value in Settings/debates without player-profile UI work.

---

## Notes

- **5 MB** enforced client-side first; `max_upload_bytes` in signature response must match **FR-004**  
- **FR-010**: No crop screen; optional circular **Image** styling only  
- Commit after each task group or logical checkpoint  
