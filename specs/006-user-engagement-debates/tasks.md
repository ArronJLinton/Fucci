# Tasks: User Engagement for AI Powered Debates

**Input**: Design documents from `specs/006-user-engagement-debates/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing. Tests are not explicitly requested in the spec; omit test-only tasks per template.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
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

- [ ] T002 Add migration to add `seeded` column (BOOLEAN NOT NULL DEFAULT false) to `comments` table in services/api/sql/schema/ (timestamped filename, e.g. 20260215000000_add_seeded_to_comments.sql)
- [ ] T003 [P] Add migration for `comment_votes` table (id, comment_id, user_id, vote_type, created_at, UNIQUE(comment_id, user_id)) with FKs to comments and users in services/api/sql/schema/
- [ ] T004 [P] Add migration for `comment_reactions` table (id, comment_id, user_id, emoji VARCHAR(20), created_at, UNIQUE(comment_id, user_id, emoji)) with FKs in services/api/sql/schema/
- [ ] T005 Add migration or seed to ensure system user (Fucci) exists: insert a dedicated user (e.g. display_name 'Fucci', identifiable by email or role) if not present in services/api/sql/schema/ or services/api/cmd/
- [ ] T006 Add sqlc queries for comment_votes (upsert/delete vote, get votes by comment_id for net score) in services/api/sql/queries/ (new file e.g. comment_votes.sql or extend existing)
- [ ] T007 Add sqlc queries for comment_reactions (insert, delete by comment_id+user_id+emoji, list by comment_id) in services/api/sql/queries/
- [ ] T008 Run sqlc generate and yarn migrate; fix any compile errors in services/api/internal/database/

**Checkpoint**: Foundation ready — user story implementation can begin

---

## Phase 3: User Story 1 — Match Debate Structure (Priority: P1) — MVP

**Goal**: Users see headline, description, and three AI-seeded starter comments (no stance labels) attributed to system user (Fucci). Seeded comments are created when a debate is generated; list comments API returns them with vote/reaction counts (public).

**Independent Test**: Generate or open a debate; call GET /api/debates/:id/comments and see three seeded comments with author (Fucci), net_score, reactions; mobile SingleDebateScreen shows comments (no stance labels).

### Implementation for User Story 1

- [ ] T009 [P] [US1] Implement GET /api/debates/{debate_id}/comments handler: list top-level comments with subcomments (one level), net_score per comment, reaction counts, user display_name and avatar_url; do not expose seeded flag; public (no auth) in services/api/internal/api/comments.go
- [ ] T010 [US1] Register GET /api/debates/{debate_id}/comments route (and mount comments router if needed) in services/api/internal/api/api.go
- [ ] T011 [US1] Extend AI prompt and debate generation to return three comment texts (agree, disagree, wildcard); after creating a debate in generate-set/create flow, insert three comments with debate_id, user_id = system user (Fucci), content from AI, seeded = true in services/api/internal/api/debates.go and services/api/internal/ai/prompt_generator.go
- [ ] T012 [US1] Add listComments(debateId) API client function in apps/mobile/src/services/api.ts
- [ ] T013 [US1] Display comments list (with net_score, reactions, author avatar/name, no stance labels) on SingleDebateScreen in apps/mobile/src/screens/SingleDebateScreen.tsx

**Checkpoint**: User Story 1 complete — users can see headline, description, and three seeded comments

---

## Phase 4: User Story 2 — Comment Interactions (Priority: P2)

**Goal**: Authenticated users can reply (top-level or subcomment, one level), upvote/downvote comments (toggle), and add/remove emoji reactions. Unauthenticated users can see vote and reaction counts but cannot engage.

**Independent Test**: As authenticated user, POST a reply, PUT vote, POST/DELETE reaction; confirm net_score and reaction counts update; as unauthenticated user, confirm list still shows counts and write attempts are rejected with 401.

### Implementation for User Story 2

- [ ] T014 [P] [US2] Implement POST /api/debates/{debate_id}/comments handler: create comment or subcomment (body content, parent_comment_id optional); enforce parent is top-level only; require auth; return DebateComment shape in services/api/internal/api/comments.go
- [ ] T015 [P] [US2] Implement PUT /api/comments/{comment_id}/vote handler: set or clear vote (body vote_type: upvote|downvote|null); require auth; return net_score in services/api/internal/api/comments.go or comment_engagement.go
- [ ] T016 [P] [US2] Implement POST /api/comments/{comment_id}/reactions and DELETE /api/comments/{comment_id}/reactions?emoji= handlers: add/toggle or remove reaction; require auth; return updated reaction counts; no max on emoji types in services/api/internal/api/comments.go or comment_engagement.go
- [ ] T017 [US2] Register POST /api/debates/{debate_id}/comments, PUT /api/comments/{comment_id}/vote, POST and DELETE /api/comments/{comment_id}/reactions routes (auth middleware for write) in services/api/internal/api/api.go
- [ ] T018 [US2] Add createComment, setCommentVote, addCommentReaction, removeCommentReaction API client functions in apps/mobile/src/services/api.ts
- [ ] T019 [US2] Add reply UI (Reply action, subcomment form) and enforce one-level subcomments in apps/mobile/src/screens/SingleDebateScreen.tsx (or new component)
- [ ] T020 [US2] Add upvote/downvote controls and net score display per comment; toggle same vote to clear in apps/mobile/src/screens/SingleDebateScreen.tsx
- [ ] T021 [US2] Add emoji reaction picker and reaction row (emoji + count) per comment; add/toggle/remove reaction in apps/mobile/src/screens/SingleDebateScreen.tsx

**Checkpoint**: User Story 2 complete — authenticated users can reply, vote, and react

---

## Phase 5: User Story 3 — Authentication Gate Modal (Priority: P2)

**Goal**: When an unauthenticated user attempts reply, vote, or reaction, show "Join the conversation" modal with Log in / Create account; after auth, return to same debate and auto-initiate the blocked action where feasible (mobile only).

**Independent Test**: As unauthenticated user, tap Reply (or Vote or React); modal appears; tap Log in, complete login; return to same debate and reply box focused or reaction picker open if feasible.

### Implementation for User Story 3

- [ ] T022 [US3] Create AuthGateModal component: title "Join the conversation", body text, primary "Log in", secondary "Create account", dismiss; overlay with darkened backdrop in apps/mobile/src/components/AuthGateModal.tsx (or in screens)
- [ ] T023 [US3] On SingleDebateScreen, when user is unauthenticated and taps Reply / Vote / React, show AuthGateModal instead of performing action; pass pending action type (reply | vote | reaction) for return state in apps/mobile/src/screens/SingleDebateScreen.tsx
- [ ] T024 [US3] On "Log in" / "Create account", navigate to Login or SignUp with return params (debateId, pendingAction); after successful auth (use AuthContext), navigate back to debate and auto-initiate pending action (e.g. focus reply input or open reaction picker) in apps/mobile/src/screens/SingleDebateScreen.tsx and apps/mobile/src/navigation/rootNavigation.ts or equivalent

**Checkpoint**: User Story 3 complete — auth gate and return-to-debate work on mobile

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and validation.

- [ ] T025 [P] Update quickstart.md with any missing steps (e.g. system user creation, comment endpoints) in specs/006-user-engagement-debates/quickstart.md
- [ ] T026 Run through quickstart.md test flows (view comments, reply, vote, reaction, auth gate, return-to-debate); fix any contract or environment gaps in apps/mobile and services/api

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — can start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 — BLOCKS all user stories
- **Phase 3 (US1)**: Depends on Phase 2 — list comments and seeded comment creation
- **Phase 4 (US2)**: Depends on Phase 2 and Phase 3 (list comments exists) — reply, vote, reaction
- **Phase 5 (US3)**: Depends on Phase 4 (write actions exist to gate) — auth gate modal and return-to-debate
- **Phase 6 (Polish)**: Depends on Phases 3–5 complete

### User Story Dependencies

- **US1 (Match Debate Structure)**: After Foundational only — independently testable (see three seeded comments)
- **US2 (Comment Interactions)**: After US1 (comments list in place) — reply/vote/react build on same comments API
- **US3 (Auth Gate)**: After US2 — gates the write actions from US2

### Parallel Opportunities

- Phase 2: T003 and T004 (two migrations) can run in parallel; T006 and T007 (two query files) can run in parallel after migrations
- Phase 3: T009 (list handler) can be done in parallel with T011 (debate generation extension) once T008 is done
- Phase 4: T014, T015, T016 (three handlers) can be implemented in parallel; T019–T021 (reply, vote, reaction UI) can be parallelized by component

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
4. Polish (quickstart validation)

### Task Summary

| Phase        | Task IDs   | Count |
|-------------|------------|-------|
| Phase 1     | T001       | 1     |
| Phase 2     | T002–T008  | 7     |
| Phase 3 US1 | T009–T013  | 5     |
| Phase 4 US2 | T014–T021  | 8     |
| Phase 5 US3 | T022–T024  | 3     |
| Phase 6     | T025–T026  | 2     |
| **Total**   |            | **26**|
