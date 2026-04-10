# Tasks: AI Powered Debate Generator

**Input**: Design documents from `/specs/004-ai-debate-generator/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.yaml

**Organization**: Tasks are grouped by user story to enable independent implementation and testing. Backend (Go) lives under `services/api/`; mobile under `apps/mobile/`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story (US1–US6) for traceability
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify feature branch and design docs; no new project creation (monorepo exists).

- [ ] T001 Verify feature branch `004-ai-debate-generator` and that design docs (spec.md, plan.md, data-model.md, research.md, contracts/api.yaml, user-stories.md) are present in specs/004-ai-debate-generator/
- [ ] T002 [P] Confirm environment: OpenAI API key, RapidAPI (API-Football) key, Redis, PostgreSQL per specs/004-ai-debate-generator/quickstart.md

---

## Phase 2: Foundational (Context Bundle — Blocking for US1, US2, US3, US4)

**Purpose**: Add H2H and league table to the context bundle so debate generation uses all three required sources. No user story work can begin until this phase is complete.

**Independent Test**: Call `GET /v1/api/debates/generate?match_id=<id>&type=pre_match` and verify response prompt content includes H2H and league table sections when fixture has league/team IDs.

- [x] T003 Extend fixture response parsing in getMatchInfo to capture league ID, season, home team ID, away team ID; add LeagueID, SeasonYear, HomeTeamID, AwayTeamID to MatchInfo in services/api/internal/api/debates.go
- [x] T004 Add LeagueID, SeasonYear, HomeTeamID, AwayTeamID to MatchDataRequest in services/api/internal/api/debate_data_aggregator.go and populate them in buildMatchDataRequest in services/api/internal/api/debates.go
- [x] T005 [P] Add HeadToHeadSummary and LeagueTableSummary (or HeadToHead/LeagueTable structs) to MatchData in services/api/internal/ai/prompt_generator.go per data-model.md
- [x] T006 Implement fetchHeadToHead(ctx, homeTeamID, awayTeamID) in services/api/internal/api/futbol.go as FetchHeadToHead; cache key h2h:{home_id}-{away_id}, TTL H2HTTL; aggregator calls Config.FetchHeadToHead
- [x] T007 Implement fetchLeagueStandings via services/api/internal/api/futbol.go GetLeagueStandingsData + FormatLeagueStandingsSummary; cache league_standings:{league_id}:{season}; aggregator calls Config.GetLeagueStandingsData
- [x] T008 In AggregateMatchData in services/api/internal/api/debate_data_aggregator.go call fetchHeadToHead when HomeTeamID and AwayTeamID are set; set matchData.HeadToHeadSummary (or equivalent); on error log and leave empty (graceful degradation)
- [x] T009 In AggregateMatchData in services/api/internal/api/debate_data_aggregator.go call fetchLeagueStandings when LeagueID and SeasonYear are set; set matchData.LeagueTableSummary (or equivalent); on error log and leave empty (graceful degradation)
- [x] T010 Add HEAD-TO-HEAD and LEAGUE TABLE sections to buildUserPrompt in services/api/internal/ai/prompt_generator.go when matchData.HeadToHeadSummary and matchData.LeagueTableSummary are present per data-model.md
- [x] T011 [P] Add unit tests for fetchHeadToHead and fetchLeagueStandings (mock HTTP) in services/api/internal/api/debate_data_aggregator_test.go or equivalent
- [x] T012 [P] Add or extend integration/unit tests for generateAIPrompt with mocked aggregator returning H2H and league table in services/api/internal/ai/prompt_generator_test.go and debate_data_aggregator_test.go to assert prompt includes both sources

**Checkpoint**: Context bundle includes H2H and league table; pre_match and post_match generation use them when available; graceful degradation when IDs or API fail.

---

## Phase 3: User Story 1 — Pre Match Debate (Priority: P1) — Epic A1

**Goal**: Fans see AI-generated pre-match debates (headline + cards) that reference H2H, table, and news; graceful degradation when sources missing.

**Independent Test**: For a fixture with status NS, GET /v1/api/debates/generate?match_id=X&type=pre_match returns 200 with headline and cards; POST /v1/api/debates/generate with debate_type=pre_match persists and returns debate. For fixture without team/league IDs, response still returns a debate (partial context).

- [x] T013 [US1] Ensure validateMatchStatusForDebateType in services/api/internal/api/debates.go rejects post_match when status is NS and pre_match when status is FT/AET/PEN per FR-006
- [x] T014 [US1] Verify generateAIPrompt and generateDebate use aggregator and prompt generator for type=pre_match; pre_match path uses buildUserPrompt with pre_match system prompt in services/api/internal/api/debates.go and services/api/internal/ai/prompt_generator.go
- [ ] T015 [US1] Add integration test: pre_match generation with mocked H2H and standings returns 200 and prompt contains expected sections in services/api/internal/api/debates_test.go or equivalent

**Checkpoint**: Pre-match debate flow end-to-end; validation and prompt content verified.

---

## Phase 3b: Generate-Set (Multi-Debate) Implementation

**Purpose**: One AI call per type returns multiple debates (e.g. 3); generate-set endpoint; cache set by (match, type); preload on match details; list UI and polling. Implements plan.md multi-debate strategy and spec.md FR-005, FR-008, FR-009.

**Goal**: Fans see multiple debates per match (pre or post); generation is one prompt per type; backend caches and returns full set; client preloads on Match Details and shows list with loading/polling.

**Independent Test**: POST /v1/api/debates/generate-set with match_id and debate_type returns 200/201 with `debates` array (e.g. 3 items); GET /v1/api/debates/match?match_id=X&debate_type=pre_match returns same set; mobile Match Details triggers background generate-set; Debate tab shows list of debates and polls when pending.

- [x] T043 [P] Extend AI prompt and response in services/api/internal/ai/prompt_generator.go: system/user prompt asks for array of N debates (default 3); increase MaxTokens (e.g. 2500–3000); parse array from OpenAI response; add GenerateDebateSetPrompt or equivalent returning []DebatePrompt
- [x] T044 Implement generateDebateSet core in services/api/internal/api/debates.go: one AI call per (match_id, debate_type) returning array; validate match status for debate_type; parse array and persist each item as debate row (cards, analytics); return full set of DebateResponse
- [x] T045 Add POST /v1/api/debates/generate-set handler in services/api/internal/api/debates.go and api.go: body GenerateDebateSetRequest (match_id, debate_type, optional count, optional force_regenerate); response GenerateDebateSetResponse (debates[], optional pending); call generateDebateSet; optional auth (no auth required per FR-010)
- [x] T046 Cache debate set in services/api/internal/api/debates.go: after generating set, store in Redis with key e.g. debates:{matchID}:{type}; on generate-set and getDebatesByMatch check cache first when returning set; TTL per plan (e.g. 24h); invalidate on force_regenerate
- [x] T047 Ensure GET /v1/api/debates/match in services/api/internal/api/debates.go accepts optional query debate_type and returns array of debates (multiple per type); filter DB result by debate_type when provided; response shape array of DebateSummary per contracts/api.yaml
- [x] T048 Add rate limit for generate-set in services/api/internal/api/debates.go: 3 requests per hour per match_id (Redis key e.g. debate_gen:{match_id} with TTL 1h); return 429 with Retry-After when exceeded per FR-008
- [x] T049 Implement deduplication for concurrent generate-set in services/api/internal/api/debates.go: one in-flight generation per (match_id, debate_type); concurrent callers wait or receive same result (e.g. sync.Map of in-flight keys or Redis lock); per spec edge case
- [x] T050 Ensure generate-set route is callable without authentication in services/api/internal/api/api.go (do not require auth middleware for POST /debates/generate-set); document optional auth and preload in specs/004-ai-debate-generator/quickstart.md
- [x] T051 [US5] Add generateDebateSet API and optional debate_type param to getDebatesByMatch in apps/mobile/src/services/api.ts; handle response debates array and optional pending from generate-set; types per contracts/api.yaml
- [x] T052 [US5] When Match Details screen mounts, trigger background call to generate-set for applicable debate type (pre_match if match not finished, post_match if finished) in apps/mobile/src/screens/MatchDetailsScreen.tsx; do not block UI
- [x] T053 [US5] Debate tab shows list of debates for current type in apps/mobile/src/screens/DebateScreen.tsx; loading until set ready; if generate-set or getDebatesByMatch returns pending, poll GET debates by match until set appears or timeout; navigate to SingleDebateScreen on item tap
- [x] T054 [P] Update specs/004-ai-debate-generator/quickstart.md with generate-set curl example, rate limit (3/hour), and preload behaviour; list GET /debates/match with optional debate_type

**Checkpoint**: Generate-set endpoint returns multiple debates; cache and GET by match/type work; mobile preloads and shows list with polling.

---

## Phase 4: User Story 2 — On-Demand Generation (Priority: P1) — Epic A2

**Goal**: When debates for match_id are missing, API triggers generation and returns pending state or minimal default; rate-limited per match.

**Independent Test**: Request debates for a match that has none; receive either 202/pending with job ID or 200 with generated/default debate within SLA; second rapid request for same match is rate-limited.

- [ ] T016 [US2] Implement on-demand trigger in getDebatesByMatch or dedicated endpoint: when no debates exist for match_id, enqueue generation job (or call sync generation) and return status pending with optional job_id in services/api/internal/api/debates.go
- [ ] T017 [US2] Add rate limit for on-demand generation per match_id (e.g. in-memory or Redis key debate_gen:{match_id} with TTL) in services/api/internal/api/debates.go
- [ ] T018 [US2] When generation is async, add polling endpoint or document polling on getDebatesByMatch until debate appears; alternatively return minimal default debate set (template) immediately per user-stories.md A2
- [ ] T019 [US2] Document on-demand behaviour and rate limit in specs/004-ai-debate-generator/quickstart.md or API contract

**Checkpoint**: On-demand generation and rate limiting working; SLA or fallback documented.

---

## Phase 5: User Story 3 — Post Match Debate (Priority: P1) — Epic A3

**Goal**: Fans see post-match debates that reference result, H2H, table impact, and news; stored separately by phase.

**Independent Test**: For a fixture with status FT, GET/POST generate with type=post_match returns 200 and debate reflects result and stats; post_match and pre_match are distinct records per match.

- [ ] T020 [US3] Ensure post_match path uses match stats and result in buildUserPrompt (post_match branch already uses final score/stats) in services/api/internal/ai/prompt_generator.go
- [ ] T021 [US3] Verify generateDebate and CreateDebate store debate_type=post_match vs pre_match; getDebatesByMatch returns both when present in services/api/internal/api/debates.go
- [ ] T022 [US3] Add integration test: post_match generation with mocked stats and H2H returns 200 and prompt includes result in services/api/internal/api/debates_test.go or equivalent

**Checkpoint**: Post-match debate flow end-to-end; phase separation verified.

---

## Phase 6: User Story 4 — Context & Degradation (Priority: P1) — Epic B1, B2

**Goal**: Context bundle is complete (match metadata, H2H, form, news); when sources are missing, debates still generate and are tagged context_quality=partial; bundle cache TTL.

**Independent Test**: Generate debate when one of H2H or league table fails; response succeeds with context_quality=partial (or equivalent); bundle cache key and TTL used.

- [ ] T023 [US4] Add context_quality field (full | partial) to debate response and optionally to DB: set partial when any of H2H, league table, or news failed or was empty in services/api/internal/api/debates.go and schema if needed
- [ ] T024 [US4] Cache context bundle or aggregated MatchData by match_id + phase with TTL in Redis (e.g. debate_ctx:{match_id}:{phase}) in services/api/internal/api/debate_data_aggregator.go or debates.go per Epic F1
- [ ] T025 [US4] Ensure AggregateMatchData never returns error when only some sources fail; return partial MatchData and let prompt builder omit missing sections in services/api/internal/api/debate_data_aggregator.go
- [ ] T026 [US4] Add unit test: aggregator with one source failing still returns MatchData with other sources populated in services/api/internal/api/debate_data_aggregator_test.go

**Checkpoint**: Graceful degradation and context_quality tagging; bundle cache in place.

---

## Phase 7: User Story 5 — Display & Engagement (Priority: P1) — Epic C1, D1, D2

**Goal**: Debates visible on match details screen (Debates tab); fans can comment and vote; auth and rate limiting for comments.

**Independent Test**: On match details screen, Debates tab shows list; empty state when no debates; user can post comment (auth) and upvote/downvote; rate limit blocks spam.

- [x] T027 [US5] Ensure match details screen has Debates tab or section that loads debates via GET /v1/api/debates/match?match_id= in apps/mobile; if none exist, trigger POST /debates/generate to create
- [x] T028 [US5] Implement or verify empty state when no debates: show “Debates generating” or “No debates yet” per Epic C1 in apps/mobile
- [ ] T029 [US5] Verify createComment and createVote endpoints require auth and persist in services/api/internal/api/debates.go; add or tighten rate limiting for POST /v1/api/debates/comments and votes per Epic D1, D2
- [ ] T030 [US5] Ensure comment and vote responses return updated counts/ranking for thread ranking (Epic D3) in services/api/internal/api/debates.go
- [x] T031 [US5] Add or update mobile UI for posting comment and casting vote on a debate thread in apps/mobile (mock data only; API logic deferred)

**Checkpoint**: Debates visible on match screen; comments and voting working with auth and rate limits.

---

## Phase 8: User Story 6 — Safety & Moderation (Priority: P1) — Epic E1, E2

**Goal**: Each debate has safety_classification (approved | needs_review | blocked); admin can approve/block; blocked debates hidden. Users can report comments; admin review queue and actions.

**Independent Test**: New debate has default classification; admin can set blocked; blocked debate not returned to clients. User can report comment; admin sees report and can delete/warn/suspend.

- [ ] T032 [US6] Add safety_classification column (or enum) to debates table and set default (e.g. needs_review or approved) on create in services/api SQL schema and database layer
- [ ] T033 [US6] Filter getDebate, getDebatesByMatch (and public list endpoints) to exclude debates where safety_classification=blocked in services/api/internal/api/debates.go
- [ ] T034 [US6] Add admin-only endpoint or action to update debate safety_classification (approve/block/edit) in services/api/internal/api/debates.go
- [ ] T035 [US6] Add comment reporting: store report (reason, optional note, comment_id, user_id) and expose in admin review queue in services/api (new table or fields + endpoint)
- [ ] T036 [US6] Implement admin actions for reported comments: delete comment, warn user, suspend/ban per Epic E2 in services/api/internal/api/debates.go or moderations module
- [ ] T037 [US6] Document moderation flows and permissions in specs/004-ai-debate-generator/user-stories.md or plan.md

**Checkpoint**: Safety classification and comment reporting; admin can block debates and act on reports.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, caching consistency, quality guardrails, and validation.

- [ ] T038 [P] Update specs/004-ai-debate-generator/quickstart.md with any new env vars or endpoints (on-demand, moderation)
- [ ] T039 Add debate quality guardrails in prompt generator: max title length, max prompt length, balanced framing; reject or retry OpenAI response when invalid per Epic A5 in services/api/internal/ai/prompt_generator.go
- [ ] T040 [P] Run quickstart validation: execute curl examples in specs/004-ai-debate-generator/quickstart.md against local API and confirm responses
- [ ] T041 Ensure Redis cache keys and TTLs for H2H, league_standings, and debate context are documented in services/api or specs/004-ai-debate-generator/data-model.md
- [ ] T042 [P] Optional: Add background job for async debate generation (enqueue in Phase 4, worker process here) per Epic F3; document in plan.md or user-stories.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately.
- **Phase 2 (Foundational)**: Depends on Phase 1. **Blocks** US1, US2, US3, US4.
- **Phase 3 (US1)**: Depends on Phase 2.
- **Phase 3b (Generate-Set)**: Depends on Phase 2 and Phase 3 (single-debate generation path); extends prompt generator and debates API for multi-debate.
- **Phase 4 (US2)**: Depends on Phase 2 (and optionally Phase 3); rate limit and on-demand behaviour align with Phase 3b (generate-set).
- **Phase 5 (US3)**: Depends on Phase 2 (and optionally Phase 3).
- **Phase 6 (US4)**: Depends on Phase 2.
- **Phase 7 (US5)**: Depends on Phase 2; may use endpoints from Phases 3, 3b, 5; list/preload from Phase 3b.
- **Phase 8 (US6)**: Depends on Phase 2; moderation applies to debates and comments from Phases 3–7.
- **Phase 9 (Polish)**: Depends on completion of desired user stories.

### User Story Dependencies

- **US1 (Pre Match)**: After Phase 2 only.
- **US2 (On-Demand)**: After Phase 2; integrates with US1/US3.
- **US3 (Post Match)**: After Phase 2 only.
- **US4 (Context & Degradation)**: After Phase 2 (extends same aggregator).
- **US5 (Display & Engagement)**: After Phase 2; needs at least one of US1/US3 for data.
- **US6 (Safety)**: After Phase 2; applies to debates and comments.

### Parallel Opportunities

- T002 can run in parallel with T001.
- T005, T011, T012 can run in parallel after T004.
- T043 can run in parallel once Phase 3 is done (prompt generator changes).
- T047, T048, T050, T054 can run in parallel after T044–T046 are done (handler/cache first).
- Phases 3, 4, 5, 6 can be worked in parallel by different developers after Phase 2; Phase 3b follows Phase 3.
- T038, T040, T041, T042 (Polish) marked [P] can run in parallel.

---

## Parallel Example: Phase 2 (Foundational)

```text
# After T004, these can run in parallel:
T005  Add HeadToHeadSummary and LeagueTableSummary to MatchData in prompt_generator.go
T006  Implement fetchHeadToHead in debate_data_aggregator.go
T007  Implement fetchLeagueStandings in debate_data_aggregator.go
T011  Unit tests for fetchHeadToHead and fetchLeagueStandings
T012  Integration tests for generateAIPrompt with H2H and league table
# Then sequentially: T008, T009 (wire into AggregateMatchData), T010 (buildUserPrompt sections).
```

### Parallel Example: Phase 3b (Generate-Set)

```text
# After T044 (generateDebateSet core), these can run in parallel:
T047  GET /debates/match optional debate_type filter
T048  Rate limit 3/hour per match_id
T050  Optional auth and quickstart doc
T054  quickstart.md generate-set curl and preload
# T045 (handler), T046 (cache), T049 (deduplication) build on T044; then T051–T053 (mobile).
```

---

## Implementation Strategy

### MVP First (Phases 1–3 + 5–6 core)

1. Complete Phase 1: Setup  
2. Complete Phase 2: Foundational (context bundle with H2H + league table)  
3. Complete Phase 3: US1 Pre Match Debate  
4. **STOP and VALIDATE**: Test pre_match generation with real or mocked fixture  
5. Complete Phase 5: US3 Post Match Debate  
6. Complete Phase 6: US4 Context & degradation (tagging, cache)  
7. Deploy/demo debate generation with three sources

### Generate-Set (Multi-Debate) Path

1. After Phase 3: Complete **Phase 3b** (T043–T054) for multi-debate: one AI call per type, generate-set endpoint, cache set, rate limit, deduplication, GET by match/type, mobile preload and list with polling.  
2. Phase 4 (US2) on-demand and rate limit can align with Phase 3b (same rate limit and optional pending/polling).  
3. Phase 7 (US5) list UI uses Phase 3b (list of debates, preload, polling).

### Incremental Delivery

1. Phase 1 + 2 → Foundation ready  
2. Phase 3 (US1) → Pre-match debates testable  
3. **Phase 3b** → Generate-set: multiple debates per type, preload, list UI  
4. Phase 4 (US2) → On-demand + rate limit  
5. Phase 5 (US3) → Post-match debates  
6. Phase 6 (US4) → Partial context and cache  
7. Phase 7 (US5) → Display and engagement  
8. Phase 8 (US6) → Moderation  
9. Phase 9 → Polish and docs  

### Suggested MVP Scope

- **Minimum**: Phase 1 + Phase 2 + Phase 3 (US1) + Phase 5 (US3) + Phase 6 (US4) — generation with three sources, pre/post, graceful degradation.  
- **With generate-set**: Add Phase 3b for multi-debate, preload, and list experience.  
- **Full MVP (spec)**: Add Phase 4 (US2), Phase 7 (US5), Phase 8 (US6) for on-demand, display/engagement, and moderation.

---

## Backlog / TODO

- [ ] **DebateDataAggregator should use Redis for news headlines**: Today `fetchNewsHeadlines` in `services/api/internal/api/debate_data_aggregator.go` calls `news.Client.FetchMatchNews` directly, so every debate context build can hit RapidAPI even when `GET /news/football/match` would be served from cache. Align with the HTTP handlers in `services/api/internal/api/news.go`: reuse `news.GenerateMatchCacheKey` (or equivalent), read through `Config.Cache` with the same TTL as match news (`cache.NewsTTL`), and only call RapidAPI on cache miss—same semantics as `/news/football/match`.

---

## Notes

- [P] tasks use different files or can be reordered without breaking dependencies.
- [USn] maps each task to the user story for traceability.
- Each user story phase is independently testable via the Independent Test criteria.
- Commit after each task or logical group; stop at checkpoints to validate.
- Paths use repository root: `services/api/`, `apps/mobile/`, `specs/004-ai-debate-generator/`.
