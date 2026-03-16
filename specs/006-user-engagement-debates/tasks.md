# Tasks: User Engagement for AI Powered Debates

**Input**: Design documents from `specs/006-user-engagement-debates/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Progress**: Phase 1 (Setup), Phase 2 (Foundational), Phase 3 (US1), Phase 4 (US2), Phase 5 (US3 — Auth Gate Modal), and Phase 6 (US4) are complete. Remaining: Phase 7 (Polish T036–T037).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing. Tests are not explicitly requested in the spec; omit test-only tasks per template. **Clarifications** (spec Session 2026-02-15) are reflected: system user via one-time migration at deploy, 500-char limit for user comments, loading/error states with retry, rate-limit comment creation only, best-effort return-to-debate auto-init. **Feature 4 (Swipe Card Voting)** clarifications: live meter = one bar for whole debate + optional per-card breakdown on tap; after all three cards voted hide stack and show meter + headline + comments; card vote final once submitted; header shows score only when available (hide score pre-match); card votes not rate-limited.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `services/api/` (Go, chi, sqlc)
- **Mobile**: `apps/mobile/src/` (React Native/Expo)
- **Migrations**: `services/api/sql/schema/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify project structure and 006 dependencies (004 debate API, 005 auth).

- [x] T001 Verify 006 dependencies and structure: 004 debate generation (services/api/internal/api/debates.go) and 005 auth (services/api/internal/api/auth.go, apps/mobile/src/context/AuthContext.tsx) exist; confirm apps/mobile and services/api layout per specs/006-user-engagement-debates/plan.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Database schema and system user that all user stories depend on. No user story work can begin until this phase is complete.

- [x] T002 Add migration to add `seeded` column (BOOLEAN NOT NULL DEFAULT false) to `comments` table in services/api/sql/schema/ (timestamped filename, e.g. 20260215000000_add_seeded_to_comments.sql)
- [x] T003 [P] Add migration for `comment_votes` table (id, comment_id, user_id, vote_type, created_at, UNIQUE(comment_id, user_id)) with FKs to comments and users in services/api/sql/schema/
- [x] T004 [P] Add migration for `comment_reactions` table (id, comment_id, user_id, emoji VARCHAR(20), created_at, UNIQUE(comment_id, user_id, emoji)) with FKs in services/api/sql/schema/
- [x] T005 Add one-time migration at deploy to ensure system user (Fucci) exists: INSERT a dedicated user (e.g. display_name 'Fucci', identifiable by email or role) if not present; migration runs before any debate generation in services/api/sql/schema/
- [x] T006 Add sqlc queries for comment_votes (upsert/delete vote, get votes by comment_id for net score) in services/api/sql/queries/ (new file e.g. comment_votes.sql or extend existing)
- [x] T007 Add sqlc queries for comment_reactions (insert, delete by comment_id+user_id+emoji, list by comment_id) in services/api/sql/queries/
- [x] T008 Run sqlc generate and yarn migrate; fix any compile errors in services/api/internal/database/

**Checkpoint**: Foundation ready — user story implementation can begin

---

## Phase 3: User Story 1 — Match Debate Structure (Priority: P1) — MVP

**Goal**: Users see headline, description, and three AI-seeded starter comments (no stance labels) attributed to system user (Fucci). Seeded comments are created when a debate is generated; list comments API returns them with vote/reaction counts (public).

**Independent Test**: Generate or open a debate; call GET /api/debates/:id/comments and see three seeded comments with author (Fucci), net_score, reactions; mobile SingleDebateScreen shows comments (no stance labels).

### Implementation for User Story 1

- [x] T009 [P] [US1] Implement GET /api/debates/{debate_id}/comments handler: list top-level comments with subcomments (one level), net_score per comment, reaction counts, user display_name and avatar_url; do not expose seeded flag; public (no auth) in services/api/internal/api/comments.go
- [x] T010 [US1] Register GET /api/debates/{debate_id}/comments route (and mount comments router if needed) in services/api/internal/api/api.go
- [x] T011 [US1] Extend AI prompt and debate generation to return three comment texts (agree, disagree, wildcard); after creating a debate in generate-set/create flow, insert three comments with debate_id, user_id = system user (Fucci), content from AI, seeded = true in services/api/internal/api/debates.go and services/api/internal/ai/prompt_generator.go
- [x] T012 [US1] Add listComments(debateId) API client function in apps/mobile/src/services/api.ts
- [x] T013 [US1] Display comments list (with net_score, reactions, author avatar/name, no stance labels) on SingleDebateScreen; include loading state while fetching and on error show user-friendly message with Retry in apps/mobile/src/screens/SingleDebateScreen.tsx

**Checkpoint**: User Story 1 complete — users can see headline, description, and three seeded comments

---

## Phase 4: User Story 2 — Comment Interactions (Priority: P2)

**Goal**: Authenticated users can reply (top-level or subcomment, one level), upvote/downvote comments (toggle), and add/remove emoji reactions. Unauthenticated users can see vote and reaction counts but cannot engage.

**Independent Test**: As authenticated user, POST a reply, PUT vote, POST/DELETE reaction; confirm net_score and reaction counts update; as unauthenticated user, confirm list still shows counts and write attempts are rejected with 401.

### Implementation for User Story 2

- [x] T014 [P] [US2] Implement POST /api/debates/{debate_id}/comments handler: create comment or subcomment (body content, parent_comment_id optional); enforce content length ≤ 500 (400 if exceeded), parent is top-level only; require auth; rate-limit comment creation per user (e.g. N per minute), return 429 when exceeded; return DebateComment shape in services/api/internal/api/comments.go
- [x] T015 [P] [US2] Implement PUT /api/comments/{comment_id}/vote handler: set or clear vote (body vote_type: upvote|downvote|null); require auth; return net_score in services/api/internal/api/comments.go or comment_engagement.go
- [x] T016 [P] [US2] Implement POST /api/comments/{comment_id}/reactions and DELETE /api/comments/{comment_id}/reactions?emoji= handlers: add/toggle or remove reaction; require auth; return updated reaction counts; no max on emoji types in services/api/internal/api/comments.go or comment_engagement.go
- [x] T017 [US2] Register POST /api/debates/{debate_id}/comments, PUT /api/comments/{comment_id}/vote, POST and DELETE /api/comments/{comment_id}/reactions routes (auth middleware for write) in services/api/internal/api/api.go
- [x] T018 [US2] Add createComment, setCommentVote, addCommentReaction, removeCommentReaction API client functions in apps/mobile/src/services/api.ts
- [x] T019 [US2] Add reply UI (Reply action, subcomment form), enforce one-level subcomments and 500-character limit (validate in UI); show loading and error with retry on submit in apps/mobile/src/screens/SingleDebateScreen.tsx (or new component)
- [x] T020 [US2] Add upvote/downvote controls and net score display per comment; toggle same vote to clear; show loading/error (and optional retry) for vote action in apps/mobile/src/screens/SingleDebateScreen.tsx
- [x] T021 [US2] Add emoji reaction picker and reaction row (emoji + count) per comment; add/toggle/remove reaction; show loading/error (and optional retry) for reaction action in apps/mobile/src/screens/SingleDebateScreen.tsx

**Checkpoint**: User Story 2 complete — authenticated users can reply, vote, and react

---

## Phase 5: User Story 3 — Authentication Gate Modal (Priority: P2)

**Goal**: When an unauthenticated user attempts reply, vote, or reaction, show "Join the conversation" modal with Log in / Create account; after auth, return to same debate and best-effort auto-initiate the blocked action (mobile only; no guarantee if state lost).

**Independent Test**: As unauthenticated user, tap Reply (or Vote or React); modal appears; tap Log in, complete login; return to same debate; optionally reply box focused or reaction picker open (best-effort).

### Implementation for User Story 3

- [x] T022 [US3] Create AuthGateModal component: title "Join the conversation", body text, primary "Log in", secondary "Create account", dismiss; overlay with darkened backdrop in apps/mobile/src/components/AuthGateModal.tsx (or in screens)
- [x] T023 [US3] On SingleDebateScreen, when user is unauthenticated and taps Reply / Vote / React, show AuthGateModal instead of performing action; pass pending action type (reply | vote | reaction) for return state in apps/mobile/src/screens/SingleDebateScreen.tsx
- [x] T024 [US3] On "Log in" / "Create account", navigate to Login or SignUp with return params (debateId, pendingAction); after successful auth (use AuthContext), navigate back to debate and best-effort auto-initiate pending action (e.g. focus reply input or open reaction picker); if state lost or init fails, showing debate only is acceptable in apps/mobile/src/screens/SingleDebateScreen.tsx and apps/mobile/src/navigation/rootNavigation.ts or equivalent

**Checkpoint**: User Story 3 complete — auth gate and return-to-debate work on mobile

---

## Phase 6: User Story 4 — Swipe Card Voting (Priority: P2)

**Goal**: Users vote on the three debate cards (agree, disagree, wildcard) by swiping right (yes) or left (no) on a stacked card UI with thumbs up/down overlay. A live debate meter at the top shows one bar for the whole debate (total yes vs no); optional per-card breakdown on tap. Header shows team badge(s) and match score when available (hide score pre-match). After voting on all three cards, hide the stack and show only meter, headline, and comments. Card vote is final once submitted; no rate limit. Unauthenticated users see cards and meter but get auth gate on swipe.

**Independent Test**: Open a debate with three cards; confirm stacked card UI, swipe right (thumbs up) / left (thumbs down), vote submitted; meter updates; after three votes stack hides. Confirm header shows team badges and score when available, no score area when pre-match. As unauthenticated user, swipe triggers auth gate.

### Implementation for User Story 4

- [x] T027 [P] [US4] Implement PUT /api/debates/{debate_id}/cards/{card_id}/vote handler: accept vote_type upvote|downvote (yes/no), require auth, one vote per user per card (replace existing row for same user+card), return CardVoteCounts (e.g. yes_count, no_count for meter); no rate limit; validate card belongs to debate in services/api/internal/api/ (e.g. card_votes.go or debates.go)
- [x] T028 [US4] Ensure GET debate (or GET debate by id) response includes card vote aggregates (total yes/no and optionally per-card yes/no) for live debate meter in services/api/internal/api/debates.go
- [x] T029 [US4] Register PUT /api/debates/{debate_id}/cards/{card_id}/vote route with auth middleware in services/api/internal/api/api.go
- [x] T030 [US4] Add setCardVote(debateId, cardId, voteType) and ensure fetchDebateById (or debate payload) returns vote counts per card and totals for meter in apps/mobile/src/services/api.ts
- [x] T031 [US4] Add header to SingleDebateScreen: team badge(s) and match score when available; when match has no score (pre-match), hide score area and show only team badges and "VS" in apps/mobile/src/screens/SingleDebateScreen.tsx
- [x] T032 [US4] Add live debate meter at top of SingleDebateScreen: one bar for whole debate (total yes vs no across all 3 cards); optional per-card breakdown on tap or in secondary view in apps/mobile/src/screens/SingleDebateScreen.tsx or apps/mobile/src/components/
- [x] T033 [US4] Build stacked card UI (3 cards layered, only top card swipeable): swipe right = yes with thumbs up overlay, swipe left = no with thumbs down overlay; on swipe call setCardVote and advance to next card in apps/mobile/src/screens/SingleDebateScreen.tsx or apps/mobile/src/components/
- [x] T034 [US4] After user has voted on all three cards, hide card stack and show only live meter, headline, and comments in apps/mobile/src/screens/SingleDebateScreen.tsx
- [x] T035 [US4] When unauthenticated user attempts swipe to vote, show AuthGateModal (reuse US3 component); pass pending action for return state in apps/mobile/src/screens/SingleDebateScreen.tsx

**Checkpoint**: User Story 4 complete — swipe card voting, meter, header, and auth gate on swipe

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and validation.

- [ ] T036 [P] Update quickstart.md with any missing steps (e.g. system user creation, comment endpoints, swipe card voting, meter, header) in specs/006-user-engagement-debates/quickstart.md
- [ ] T037 Run through quickstart.md test flows (view comments, reply, vote, reaction, auth gate, return-to-debate, swipe card voting, meter, header); fix any contract or environment gaps in apps/mobile and services/api

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — can start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 — BLOCKS all user stories
- **Phase 3 (US1)**: Depends on Phase 2 — list comments and seeded comment creation
- **Phase 4 (US2)**: Depends on Phase 2 and Phase 3 (list comments exists) — reply, vote, reaction
- **Phase 5 (US3)**: Depends on Phase 4 (write actions exist to gate) — auth gate modal and return-to-debate
- **Phase 6 (US4)**: Depends on Phase 2 (votes table for cards) and Phase 3 (debate/cards from API); auth gate on swipe reuses US3 modal (Phase 5)
- **Phase 7 (Polish)**: Depends on Phases 3–6 complete

### User Story Dependencies

- **US1 (Match Debate Structure)**: After Foundational only — independently testable (see three seeded comments)
- **US2 (Comment Interactions)**: After US1 (comments list in place) — reply/vote/react build on same comments API
- **US3 (Auth Gate)**: After US2 — gates the write actions from US2
- **US4 (Swipe Card Voting)**: After Phase 2 (votes table) and Phase 3 (debate + cards); swipe auth gate reuses US3 (Phase 5) when implemented

### Parallel Opportunities

- Phase 2: T003 and T004 (two migrations) can run in parallel; T006 and T007 (two query files) can run in parallel after migrations
- Phase 3: T009 (list handler) can be done in parallel with T011 (debate generation extension) once T008 is done
- Phase 4: T014, T015, T016 (three handlers) can be implemented in parallel; T019–T021 (reply, vote, reaction UI) can be parallelized by component
- Phase 6: T027 (card vote handler) and T028 (GET debate vote aggregates) can be done in parallel; T031 (header), T032 (meter), T033 (stack) can be parallelized by component where dependencies allow

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (migrations, system user, sqlc)
3. Complete Phase 3: User Story 1 (list comments, seeded comments in debate gen, display on SingleDebateScreen)
4. **STOP and VALIDATE**: Open a debate and see headline, description, three seeded comments (Fucci), with counts
5. Deploy/demo if ready

### Incremental Delivery

1. Foundation → US1 (see seeded comments) → MVP
2. US2 (reply, vote, react) → test independently
3. US3 (auth gate, return-to-debate) → test independently
4. US4 (swipe card voting, meter, header) → test independently (stack, meter, header, card vote API)
5. Phase 7 Polish (quickstart validation including swipe flows)

### Task Summary

| Phase        | Task IDs    | Count |
|-------------|-------------|-------|
| Phase 1     | T001        | 1     |
| Phase 2     | T002–T008   | 7     |
| Phase 3 US1 | T009–T013   | 5     |
| Phase 4 US2 | T014–T021   | 8     |
| Phase 5 US3 | T022–T024   | 3     |
| Phase 6 US4 | T027–T035   | 9     |
| Phase 7     | T036–T037   | 2     |
| **Total**   |             | **35**|
