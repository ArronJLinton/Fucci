# Implementation Plan: AI Powered Debate Generator

**Branch**: `004-ai-debate-generator` | **Date**: 2025-02-15 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/004-ai-debate-generator/spec.md`

## Summary

Generate AI-powered debate prompts for football matches (Pre Match and Post Match) using head-to-head history, league table, and news articles. Support multiple debates per type (e.g. 3 per type), preload generation when the user opens match details, and deliver via a dedicated Debate tab with single-debate drill-down. One AI call per type returns multiple debates in a single response to control cost and latency; cache and DB store a set per (match_id, debate_type).

## Technical Context

**Language/Version**: Go 1.22+ (backend), TypeScript / React Native (mobile)  
**Primary Dependencies**: chi router, OpenAI API, API-Football (RapidAPI), Redis, PostgreSQL  
**Storage**: PostgreSQL (debates, debate_cards, debate_analytics, comments, votes); Redis (context and prompt caching)  
**Testing**: Go tests (unit/integration) in `services/api`; Jest/React Native in `apps/mobile`  
**Target Platform**: Backend (Linux/server); Mobile (iOS/Android via Expo)  
**Project Type**: Monorepo (mobile app + API); `services/api`, `apps/mobile`  
**Performance Goals**: Debate generation &lt; 10–20s (sync) or non-blocking (async); API p95 &lt; 200ms for read endpoints  
**Constraints**: OpenAI token budget; rate limits on external APIs (news, API-Football); graceful degradation when sources fail  
**Scale/Scope**: Multiple debates per match (e.g. 3 pre + 3 post); cache by match + type; list UI and single-debate screen

## Multi-debate generation and preload (implementation approach)

The following approach is adopted for generating **multiple debates per pre- and post-match** and for **load time / UX**.

### 1. One prompt per type returning multiple debates

- **Decision**: Use **one AI call per debate type** (pre_match, post_match). Each call returns **multiple debates** (e.g. 3) in a **single JSON response** (array of debate prompts).
- **Rationale**: Fewer round-trips, lower token usage than N separate calls; avoids extra rate-limit pressure; cost stays in the “fraction of a cent per match” range.
- **Implementation**: Extend the prompt and response contract so the model returns an array of debates (each with headline, description, cards). Increase `MaxTokens` as needed (e.g. 2500–3000) for the larger output. Parse and persist each item as a separate debate row.

### 2. Preload when match details open

- **Decision**: When the user **opens Match Details** (match screen), trigger **background generation** for the applicable debate type (pre_match if match not finished, post_match if finished). Do not block the UI; user can browse Lineup, Table, News while debates generate.
- **Behaviour**:
  - When the user opens the **Debate** tab: if the result is already in cache/DB → show it immediately. If still generating → show one loading state until the full set is ready.
- **Rationale**: First-creation load time is hidden for most users; no progressive rendering required for v1.

### 3. Backend shape

| Concern | Decision |
|--------|----------|
| **Cache** | One key per (match, type) storing an **array** of debates, e.g. `pre_match_debates:{matchID}` or `debates:{matchID}:{type}`. Simpler than one key per debate index. |
| **DB** | Keep **one row per debate**; multiple rows per `(match_id, debate_type)` is expected and matches “multiple debates per type.” No schema change to uniqueness; list endpoints return all debates for the match (optionally filtered by type). |
| **API** | Add a **“generate set”** endpoint (or extend existing generate) that returns **multiple debates at once** (e.g. 3). Client and list UI treat debates as a **list** (e.g. show first as primary or show all in a list). |

### 4. Client behaviour

- **List**: Debate tab shows the set of debates for the current type (pre or post). Each item can open SingleDebateScreen (existing behaviour).
- **Loading**: Single loading state until the full set is ready; no progressive “debates appear one by one” in v1 unless specified later.

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

**Code Quality Standards:**

- [x] TypeScript strict mode compliance verified (apps/mobile)
- [x] ESLint configuration defined with zero warnings
- [x] Function complexity ≤ 10, length ≤ 50 lines (target)
- [x] Meaningful naming conventions established

**Testing Standards:**

- [x] TDD approach planned for new features
- [x] Unit test coverage target ≥ 80% identified
- [x] Integration test requirements defined (API, aggregator, prompt generator)
- [x] E2E test scenarios for P1 user stories planned (Debate tab, single debate)

**User Experience Consistency:**

- [x] Design system compliance verified
- [x] Accessibility requirements (WCAG 2.1 AA) identified
- [x] Loading states and error handling planned (empty, generating, error)
- [x] Responsive design considerations documented (mobile-first)

**Performance Requirements:**

- [x] Performance benchmarks defined (debate generation SLA; API latency)
- [x] Bundle size impact assessed (mobile)
- [x] Database query performance targets set (indexes on match_id, debate_type)
- [x] Caching strategy planned (Redis by match+type; preload on match details)

**Developer Experience:**

- [x] Documentation requirements identified (quickstart, API contract)
- [x] API documentation needs defined (OpenAPI in contracts/)
- [x] Development environment setup documented (quickstart.md)
- [x] Code review guidelines established (constitution)

## Project Structure

### Documentation (this feature)

```text
specs/004-ai-debate-generator/
├── plan.md              # This file
├── research.md          # Phase 0 output (sources, prompt, multi-debate)
├── data-model.md        # Phase 1 output (entities, cache, multi-debate flow)
├── quickstart.md        # Phase 1 output (env, endpoints, examples)
├── contracts/           # API contract (generate, generate-set, list)
│   └── api.yaml
├── user-stories.md      # Epics A–H
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
services/api/
├── internal/
│   ├── api/
│   │   ├── debates.go         # Handlers (generate, generate-set, get by match/id)
│   │   └── debate_data_aggregator.go
│   ├── ai/
│   │   └── prompt_generator.go  # One prompt per type → multiple debates (response array)
│   └── news/
│       └── client.go
├── sql/schema/                # debates, debate_cards, debate_analytics migrations
└── cmd/

apps/mobile/
├── src/
│   ├── screens/
│   │   ├── MatchDetailsScreen.tsx   # Trigger preload when mounted
│   │   ├── DebateScreen.tsx         # List of debates for type; loading state
│   │   └── SingleDebateScreen.tsx   # Single debate with cards and comments
│   └── services/
│       └── api.ts                   # getDebatesByMatch, generateDebateSet, fetchDebateById
```

**Structure Decision**: Monorepo with Go API (`services/api`) and React Native app (`apps/mobile`). Debate generation and preload are implemented in the API; mobile triggers preload on Match Details and consumes list/single-debate endpoints.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| None currently | — | — |
