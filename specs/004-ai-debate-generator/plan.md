# Implementation Plan: AI Powered Debate Generator

**Branch**: `004-ai-debate-generator` | **Date**: 2025-02-15 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/004-ai-debate-generator/spec.md`

## Summary

Implement and document the AI Powered Debate Generator so it explicitly uses three required sources—**head-to-head history**, **league table**, and **news articles**—for both **Pre Match** and **Post Match** debate types, and supports the full scope in [spec.md](./spec.md) and [user-stories.md](./user-stories.md) (Epics A–H). The codebase already has debate generation (handlers, aggregator, OpenAI prompt generator); this plan aligns the implementation with the spec by guaranteeing H2H and league table are fetched and passed into the AI prompt, and by documenting behaviour and contracts.

**Technical approach**: Extend the existing `DebateDataAggregator` and AI `MatchData`/prompt builder to include H2H and league standings; add or reuse API-Football endpoints for H2H and league table; keep news as already implemented; ensure pre/post match rules and graceful degradation are explicit. Phased work: core generation + context (Epics A, B) first; then delivery (C), engagement (D), safety (E), cost/performance (F), and admin (G) per MVP cut in the spec.

## Technical Context

**Language/Version**: Go 1.22+, TypeScript (React Native) for consumers  
**Primary Dependencies**: Existing Fucci API (chi router), OpenAI API (gpt-4o-mini), API-Football (RapidAPI) for fixtures/standings/H2H, internal news client (RapidAPI Real-Time News), PostgreSQL (debates storage), Redis (caching)  
**Storage**: PostgreSQL (debates, debate_analytics); Redis for caching external API responses (news, standings, H2H as needed)  
**Testing**: Go testing (unit + integration for handlers/aggregator), mock OpenAI and football API  
**Target Platform**: Backend API (Linux/server); mobile app consumes debate API  
**Project Type**: Mobile + API (existing monorepo: `apps/mobile`, `services/api`)  
**Performance Goals**: Debate generation &lt; 30s end-to-end; API p95 &lt; 200ms for read endpoints; aggregate external calls in parallel where possible  
**Constraints**: All three sources (H2H, league table, news) required by spec; graceful degradation when a source fails; no PII in prompts  
**Scale/Scope**: Same as 001-football-community (10k users); feature scope: 2 debate types, 3 mandatory sources, context bundle (H2H, form, stats, news), debate generation + delivery + engagement (comments, voting) + moderation + caching/async; see [user-stories.md](./user-stories.md) for Epics A–H and MVP cut

**Current state**:

- `services/api/internal/api/debates.go`: `generateAIPrompt`, `generateDebate`; pre_match/post_match validation; `DebateDataAggregator` used.
- `services/api/internal/api/debate_data_aggregator.go`: Fetches lineups, match stats, news headlines, social sentiment. **Does not** fetch head-to-head history or league table.
- `services/api/internal/ai/prompt_generator.go`: `MatchData` has NewsHeadlines, Stats, Lineups, SocialSentiment; **no H2H or LeagueTable** fields yet.
- League standings: `getLeagueStandingsByLeagueId`, `getLeagueStandingsByTeamId` exist in `futbol.go`; aggregator does not call them.
- H2H: No head-to-head endpoint in codebase; API-Football supports H2H (e.g. `fixtures/headtohead?h2h=id1-id2`); to be added.

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

**Code Quality Standards:**

- [x] TypeScript strict mode compliance (mobile app existing)
- [x] ESLint zero warnings (existing)
- [x] Function complexity ≤ 10, length ≤ 50 lines (enforced in changes)
- [x] Meaningful naming (H2H, league table, debate type)

**Testing Standards:**

- [ ] TDD for new aggregator fetchers (H2H, league table) and prompt inclusion
- [ ] Unit test coverage ≥ 80% for new/updated debate logic
- [ ] Integration tests: debate generation with mocked OpenAI and football API
- [ ] E2E: P1 user stories (pre-match and post-match debate flow) covered in test plan

**User Experience Consistency:**

- [x] Design system (existing debate UI in app)
- [x] Loading and error states (existing patterns)
- [x] Graceful degradation (spec FR-007)

**Performance Requirements:**

- [x] Latency target for debate generation documented (e.g. &lt; 30s)
- [x] Caching: reuse Redis for standings; add H2H cache where appropriate
- [x] No new mobile bundle impact (API-only changes)

**Developer Experience:**

- [x] API contracts in `contracts/`; quickstart for local debate generation
- [x] Document which endpoints/sources are used

## Project Structure

### Documentation (this feature)

```text
specs/004-ai-debate-generator/
├── plan.md              # This file
├── spec.md              # Feature spec (scope, roles, epics overview, FRs)
├── user-stories.md      # Full epics A–H with user stories and acceptance criteria
├── research.md          # Phase 0: H2H/standings APIs, prompt design
├── data-model.md        # Phase 1: entities, context bundle, data flow
├── quickstart.md        # Phase 1: run and test debate generation
├── contracts/           # Phase 1: OpenAPI for debate endpoints
│   └── api.yaml
└── tasks.md             # Phase 2 (/speckit.tasks) – not created by /speckit.plan
```

### Source Code (repository root)

```text
services/api/
├── internal/
│   ├── api/
│   │   ├── debates.go           # Handlers: generateAIPrompt, generateDebate
│   │   ├── debate_data_aggregator.go  # Add H2H + league table fetch
│   │   └── ...
│   ├── ai/
│   │   └── prompt_generator.go  # MatchData + buildUserPrompt: add H2H, LeagueTable
│   └── ...
└── ...

apps/mobile/
└── ...   # Consumes debate API; no structural change required for this spec
```

**Structure decision**: Backend-only changes under `services/api`; mobile app continues to call existing debate endpoints. New aggregator fetchers and AI types live in existing packages.

## Complexity Tracking

| Item              | Why needed                         | Simpler alternative rejected   |
|-------------------|------------------------------------|--------------------------------|
| H2H external API  | Spec requires head-to-head history | Hardcoding fake H2H not acceptable |
| League table fetch in aggregator | Spec requires league table as source | Omitting table would violate FR-003 |

## Phases (from /speckit.plan)

- **Phase 0**: Research H2H API (API-Football), league table integration, prompt design for H2H + table + news. Output: `research.md`.
- **Phase 1**: Data model (entities, aggregator inputs/outputs), contracts (OpenAPI), quickstart. Output: `data-model.md`, `contracts/api.yaml`, `quickstart.md`. Re-check constitution after design.
- **Phase 2**: Implementation tasks (e.g. add H2H fetch, league table to aggregator; extend MatchData and prompt; tests). Output: `tasks.md` via `/speckit.tasks` (not part of this command).

## Risks and Mitigations

- **External API limits**: H2H and standings calls add to RapidAPI usage. Mitigation: Cache H2H and standings with TTL (e.g. 6–12h) in Redis.
- **Missing league_id on fixture**: Some fixtures may not have league/season. Mitigation: Resolve from fixture response; if absent, skip league table and document in response/debug.

## Epics Reference

Full user stories and acceptance criteria are in [user-stories.md](./user-stories.md). MVP prioritises: **A1, B1, C1** (generate + context + display), **D1, D2** (comment + vote), **E1, E2** (moderation), **F1, F3** (caching + async). Epics G (admin/observability) and H (future) are post-MVP or phased.

## Acceptance Criteria (from spec)

- Pre Match and Post Match debates use all three sources when available (SC-001).
- Graceful degradation when a source fails (FR-007, SC-003).
- Debate prompt structure (headline + cards, optional side labels/topic tags) supports client display (FR-005).
- AI content guidelines from 001-football-community remain applicable (SC-004).
- Caching (F1), on-demand generation with rate limiting (A2, F2), and moderation (E1, E2) as per user-stories.
