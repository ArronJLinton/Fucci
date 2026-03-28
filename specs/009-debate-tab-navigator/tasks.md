# Tasks: Debate Tab & Main Debates Experience

**Feature dir** (repo-relative from monorepo root): `specs/009-debate-tab-navigator/`  
**Input**: Design documents from that directory, aligned with **[plan.md](./plan.md)** (2026-03-28).

**Plan constraints (must hold)**:

- **Debate topics**: Main-feed debates are **grounded in world football news/headlines** via **[004-ai-debate-generator](../004-ai-debate-generator/spec.md)** context bundles — **not** via a new news crawler or ingestion layer in **009** ([plan.md](./plan.md) Summary, Technical Context).
- **009 scope**: Public + authenticated **feed APIs**, sqlc, and mobile **MainDebatesScreen** / navigation; optional **`source_headline` / `source_url` / `source_published_at`** on DTOs when DB/API exposes them ([data-model.md](./data-model.md), [contracts/debates-feed.yaml](./contracts/debates-feed.yaml)).
- **004 coordination**: Weight **top football headlines** in generation jobs — tracked in **T025** ([plan.md](./plan.md) Next Steps, [research.md](./research.md) §7).

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [data-model.md](./data-model.md), [contracts/debates-feed.yaml](./contracts/debates-feed.yaml), [research.md](./research.md), [quickstart.md](./quickstart.md)

**Tests**: Plan and constitution call for Go tests on feed SQL/handlers; no separate test phase for RN unless adding new tests—optional noted below.

**Organization**: Phases follow user stories in **implementation dependency order** (US4 before US5 so Debates tab → detail works before detail guest/auth polish).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no blocking incomplete dependencies)
- **[Story]**: `[US1]` … `[US5]` for user-story phases only
- Paths are repo-relative from monorepo root `Fucci/`

## Path Conventions (this monorepo)

- **API**: `services/api/`
- **Mobile**: `apps/mobile/`

## Completion status

| Scope | Tasks | Status |
|-------|-------|--------|
| Setup | T001–T002 | Done |
| Foundational (feeds API) | T003–T007 | Done |
| US1 Debates tab + feed | T008–T011 | Done |
| US2 New vs activity + guest CTA | T012–T014 | Done |
| US3 Hero swipe + card vote + auth | T015–T017 | Done |
| US4 List → detail navigation | T018–T019 | Done |
| US5 Detail comments + guest read-only | T020–T021 | Done |
| Polish & cross-cutting | T022–T025 | Done |

**Progress**: **25 / 25** tasks checked off below.

---

## Phase 1: Setup (shared prerequisites)

**Purpose**: Dependencies and tooling for swipe UX and sqlc workflow.

- [x] T001 Add `react-native-gesture-handler` and `react-native-reanimated` with Expo-compatible versions in `apps/mobile/package.json` (e.g. `npx expo install react-native-gesture-handler react-native-reanimated`); add Reanimated Babel plugin to `apps/mobile/babel.config.js` per Expo docs if missing.
- [x] T002 [P] Confirm `services/api/sqlc.yaml` (or project sqlc config) includes `sql/queries/debates.sql` and document regenerate command for implementers.

**Checkpoint**: Mobile ready for gesture work; API ready for new sqlc queries.

---

## Phase 2: Foundational (blocking—all feed work)

**Purpose**: `GET /v1/api/debates/public-feed` (no auth) and `GET /v1/api/debates/feed` (JWT) per [contracts/debates-feed.yaml](./contracts/debates-feed.yaml). **Do not** add news fetching/RSS/crawler code here — headline provenance fields are optional pass-through when stored (004/migrations). **No user story work until this phase completes.**

- [x] T003 Add sqlc queries in `services/api/sql/queries/debates.sql` for (1) public list ordered by engagement score desc, `created_at` desc, capped by `limit`; (2) per-user `new_debates` / `voted_debates` using **one vote per debate** on any **agree**/**disagree** card (see [data-model.md](./data-model.md)). Include **optional** provenance columns in SELECTs **only if** they exist on `debates` (or joined table); otherwise omit and map as null in Go until a migration adds them.
- [x] T004 Implement response structs and handlers `getDebatesPublicFeed` / `getDebatesFeed` in `services/api/internal/api/debates.go`, reusing or extending `services/api/internal/api/debate_data_aggregator.go` for `DebateSummary` + analytics mapping as needed; populate optional **`source_headline`**, **`source_url`**, **`source_published_at`** in JSON when data exists, per [contracts/debates-feed.yaml](./contracts/debates-feed.yaml) (omit keys or null per existing API JSON conventions).
- [x] T005 Register `GET /debates/public-feed` (no `RequireAuth`) and `GET /debates/feed` (with `auth.RequireAuth`) on `debateRouter` in `services/api/internal/api/api.go`—order routes so `/feed` and `/public-feed` are registered before `/{id}` if needed.
- [x] T006 Run `sqlc generate` from `services/api/`, fix compile errors in `services/api/internal/database/`, and ensure `go build` passes for `services/api`.
- [x] T007 Add tests in `services/api/internal/api/debates_test.go` (and/or query-focused tests) covering public-feed ordering shape and authenticated feed bucket split.

**Checkpoint**: `curl` against public-feed without token; feed with `Authorization` returns two buckets—unblocks all mobile stories.

---

## Phase 3: User Story 1 — Open Debates from bottom tabs (Priority: P1)

**Goal**: Debates tab visible; main screen loads public feed when logged out and user feed when logged in.

**Independent test**: Tap **Debates** tab → `MainDebatesScreen` mounts → loading then data or error; no match navigation required.

- [x] T008 [P] [US1] Add TypeScript types for `PublicDebateFeedResponse` and `DebateFeedResponse` in `apps/mobile/src/types/debate.ts` aligned with `contracts/debates-feed.yaml`, including **optional** `source_headline`, `source_url`, `source_published_at` on debate summary types.
- [x] T009 [US1] Add `fetchDebatesPublicFeed` / `fetchDebatesFeed` (base URL + auth header from existing auth pattern) in `apps/mobile/src/services/debate.ts`.
- [x] T010 [US1] Create `apps/mobile/src/screens/MainDebatesScreen.tsx` using TanStack Query: choose public vs authenticated endpoint based on `AuthContext` session; loading and error UI; **pull-to-refresh** refetch per spec FR-007.
- [x] T011 [US1] Add `Debates` to `MainTabParamList` in `apps/mobile/src/types/navigation.ts`; implement `DebatesStack` in `apps/mobile/App.tsx` (native stack: `MainDebates` → `SingleDebate` mirroring `HomeStack` pattern); add tab icon/entry for Debates.

**Checkpoint**: Signed-in and signed-out users both get a working feed fetch on the new tab.

---

## Phase 4: User Story 2 — New vs voted sections (Priority: P1)

**Goal**: **NEW DEBATES** block above **MY ACTIVITY**; empty states; guest **My Activity** = empty state + sign-in CTA.

**Independent test**: Mock or staging API with both buckets → order correct; one empty bucket → layout holds; guest → second section shows CTA.

- [x] T012 [US2] Render **new** above **voted** in `apps/mobile/src/screens/MainDebatesScreen.tsx` (`SectionList` or paired `FlatList`s + `keyExtractor`); render **optional secondary source line** (FR-009) when `source_headline` / `source_url` is present on a row; **headline-only** when absent (no reserved empty space).
- [x] T013 [US2] Empty states for empty `new_debates` or `voted_debates` (copy + optional CTA) in `MainDebatesScreen.tsx` — including **no hero swipe** when `new_debates` (or guest `debates`) is empty even if `voted_debates` has items (per spec).
- [x] T014 [US2] For guests (`public-feed` path): keep **MY ACTIVITY** header with empty state + **sign-in CTA** navigating to `Login` (reuse patterns from `SingleDebateScreen` / auth flows) in `MainDebatesScreen.tsx`.

**Checkpoint**: IA matches spec FR-003 for signed-in and guest.

---

## Phase 5: User Story 3 — Swipe to vote on top debate card (Priority: P1)

**Goal**: Tinder-style swipe on featured card; right = agree (upvote), left = disagree (downvote); auth gate when logged out.

**Independent test**: Logged-in swipe calls `PUT /v1/api/debates/{debateId}/cards/{cardId}/vote`; logged-out swipe opens auth flow.

- [x] T015 [US3] Extract or implement top-card stack UI with `react-native-gesture-handler` + `react-native-reanimated` (e.g. `apps/mobile/src/components/DebateHeroSwipeCard.tsx`) using **first** debate in `new_debates[0]` or guest `debates[0]` (first card of that debate—swipe still gates vote when logged out). Show **optional** provenance line on hero when `source_*` fields present (FR-009). **If `new_debates` / guest `debates` is empty, render no hero** (no fallback from `voted_debates` per spec).
- [x] T016 [US3] On swipe completion, call existing card-vote API from `apps/mobile/src/services/debate.ts` with correct `debateId` / `cardId` / `vote_type` (upvote/downvote per spec).
- [x] T017 [US3] When unauthenticated, intercept swipe to run **auth gate** (same `returnToDebate` / pending pattern as 006 if applicable) in `MainDebatesScreen.tsx` or hero component.

**Checkpoint**: Swipe UX matches spec clarifications and 006 semantics.

**Mobile note (RNGH)**: `GestureDetector` and the Gesture API must sit under **`GestureHandlerRootView`**. The root layout in `apps/mobile/App.tsx` wraps the navigation tree with `style={{ flex: 1 }}` so `DebateHeroSwipeCard` and any other gesture surfaces work at runtime.

---

## Phase 6: User Story 4 — Open debate detail (Priority: P2)

**Goal**: Tap voted row or public browse row → `SingleDebateScreen` with `match` + `debate` params; guest sees read-only detail.

**Independent test**: From Debates tab, tap list row → detail opens with correct IDs; guest can read headline/meter/comments.

- [x] T018 [US4] Wire `onPress` on **MY ACTIVITY** rows in `MainDebatesScreen.tsx` to `navigation.navigate('SingleDebate', { match, debate })` within Debates stack (fetch full debate if list only has summary—reuse existing `getDebate` client if needed).
- [x] T019 [US4] Wire `onPress` on public browse rows for guests to the same `SingleDebate` route with params loaded from list + `getDebate` as needed in `MainDebatesScreen.tsx`.

**Checkpoint**: Debates tab → detail navigation works for both personas.

---

## Phase 7: User Story 5 — Comments & engagement on detail (Priority: P1)

**Goal**: Authenticated users use 006 comment APIs; guests read-only; **no AI analysis strip** (FR-006c).

**Independent test**: Guest on detail: thread visible; post/reply/comment-vote triggers auth; signed-in user flows unchanged from 006.

- [x] T020 [US5] Remove or do not add any **AI analysis strip** UI in `apps/mobile/src/screens/SingleDebateScreen.tsx` (verify no stray AI block remains); show optional **`source_headline` / `source_url`** line on detail when present in debate payload (same rule as main feed per FR-009).
- [x] T021 [US5] Ensure guest mode on `SingleDebateScreen.tsx`: read-only thread; composer and comment vote actions use auth gate consistent with existing `AuthPendingAction` / login return flow.

**Checkpoint**: FR-006, FR-006b, FR-006c satisfied on detail.

---

## Phase 8: Polish & cross-cutting

**Purpose**: Docs, a11y, CI-style checks.

- [x] T022 [P] Update `specs/009-debate-tab-navigator/quickstart.md` with unauthenticated `curl` for `GET /v1/api/debates/public-feed?limit=30` alongside existing feed example.
- [x] T023 [P] Add accessibility labels to Debates tab in `apps/mobile/App.tsx` and main list/swipe regions in `MainDebatesScreen.tsx` / hero component.
- [x] T024 Run `go test ./...` in `services/api` and `yarn type-check` (or `tsc --noEmit`) in `apps/mobile`; fix regressions.
- [x] T025 [P] Per [plan.md](./plan.md) Next Steps: align **[004-ai-debate-generator](../004-ai-debate-generator/spec.md)** jobs/context bundle so generation emphasizes **top / trending world football news and headlines** (see `specs/009-debate-tab-navigator/research.md` §7). Implement or verify in **004** (and migrations for optional `source_*` on `debates` if product wants provenance in feeds); **009** does not own ingestion—document outcome in PR notes.

---

## Dependencies & execution order

### Phase dependencies

| Phase | Depends on |
|-------|------------|
| Phase 1 Setup | — |
| Phase 2 Foundational | Phase 1 (T001 optional before mobile UI; T002 anytime) |
| Phase 3 US1 | Phase 2 |
| Phase 4 US2 | Phase 3 |
| Phase 5 US3 | Phase 4 (needs sections + hero target debate) |
| Phase 6 US4 | Phase 4 (lists must exist); can overlap with Phase 5 if rows navigable before swipe polish |
| Phase 7 US5 | Phase 6 recommended (test detail from Debates tab) |
| Phase 8 Polish | Phases 3–7 as applicable |

### User story dependency notes

- **US1** → **US2** → **US3**: sequential on `MainDebatesScreen.tsx`.
- **US4** (P2): Depends on list UI from US2; should precede **US5** for end-to-end Debates → detail testing.
- **US5** (P1): Touches `SingleDebateScreen.tsx`; can start after US4 or in parallel once detail is reachable from Home/Match, but validate on Debates path after US4.

### Parallel opportunities

- **T008** [P] types and **T002** [P] sqlc audit can run alongside **T001** before Phase 2.
- After Phase 2: **T022** / **T023** can run late in parallel.
- **T020** / **T021** touch one screen file—sequential within US5.

### Parallel example: Phase 2 + US1 bootstrap

```bash
# After T003: sql reviewer can work while implementer starts DTO sketches — then T004–T006 serial.
# After Phase 2:
# Parallel: T008 (types) while reviewing T009 service signature.
```

---

## Implementation strategy

### MVP (minimum shippable)

1. Complete **Phase 2** (API feeds).
2. Complete **Phase 3** (tab + fetch only)—**User Story 1** standalone demo.

### Incremental delivery

1. Phase 1 → Phase 2 → **Phase 3 (US1)** → validate tab + feed.
2. **Phase 4 (US2)** → validate IA + guest CTA + optional source lines.
3. **Phase 5 (US3)** → validate swipe + votes + hero selection rules.
4. **Phase 6 (US4)** → validate navigation.
5. **Phase 7 (US5)** → validate detail + FR-006c + FR-009 on detail.
6. **Phase 8** → polish + **T025** (004 headline weighting / optional DB fields for provenance).

### Plan vs 009 boundaries

| Owner | Responsibility |
|-------|----------------|
| **004** | News/article context for AI debate generation; optional DB fields for headline provenance. |
| **009** | Feed endpoints, mobile tab, lists, hero, swipe UX, pass-through of optional `source_*` in API + UI when data exists. |
| **006** | Auth-gated votes, comments, comment votes (unchanged). |

### Suggested MVP scope

- **User Story 1** only after Foundation: Debates tab + dual feed fetch + basic list placeholder proves connectivity (expand to US2 immediately after for usable UI).

---

## Task summary

| Phase | Task IDs | Count |
|-------|----------|------:|
| Setup | T001–T002 | 2 |
| Foundational | T003–T007 | 5 |
| US1 | T008–T011 | 4 |
| US2 | T012–T014 | 3 |
| US3 | T015–T017 | 3 |
| US4 | T018–T019 | 2 |
| US5 | T020–T021 | 2 |
| Polish | T022–T025 | 4 |
| **Total** | **T001–T025** | **25** |

### Per user story (labels)

| Story | Tasks | Count |
|-------|-------|------:|
| US1 | T008–T011 | 4 |
| US2 | T012–T014 | 3 |
| US3 | T015–T017 | 3 |
| US4 | T018–T019 | 2 |
| US5 | T020–T021 | 2 |

**Format validation**: Tasks use `- [ ] Tnnn …` when pending and `- [x] Tnnn …` when done; each description includes at least one file path.
