# Implementation Plan: Debate Tab & Main Debates Experience

**Branch**: `009-debate-tab-navigator` | **Date**: 2026-03-28 | **Spec**: [spec.md](./spec.md)

## Summary

Add a **Debates** tab to the Expo bottom navigator with a **main debates** screen: **new** (incomplete card-vote) debates on top, **voted** debates below; **Tinder-style** swipes on the featured card (**right = agree/upvote**, **left = disagree/downvote**) wired to existing card-vote APIs; tap **My Activity** rows **or** (for **guests**) a public browse row to open **debate detail** (`SingleDebate`) — **guests** read headline, meter, and comments; **signed-in** users get full engagement per **006**.

Backend adds:

- **`GET /v1/api/debates/public-feed`** (or equivalent) — **no auth**, read-only debate summaries for **guests** (browse).
- **`GET /v1/api/debates/feed`** — **authenticated**, returns `new_debates` and `voted_debates` for the current user.

See [research.md](./research.md), [contracts/debates-feed.yaml](./contracts/debates-feed.yaml). All **writes** (card vote, comments, comment votes) remain on existing 006 routes behind auth.

## Technical Context

**Language/Version**: Go 1.22+ (API), TypeScript strict (Expo / React Native 0.81+)  
**Primary Dependencies**: chi router, sqlc-generated queries, React Navigation 7, TanStack React Query (existing)  
**Storage**: PostgreSQL (`debates`, `debate_cards`, `votes`, `debate_analytics`, …)  
**Testing**: `go test ./...`, Jest/RN tests for UI where applicable; integration tests for new feed query  
**Target Platform**: iOS/Android via Expo 54  
**Project Type**: Monorepo — `services/api` + `apps/mobile`  
**Performance Goals**: Feed p95 &lt; 200ms (constitution); list virtualization for long feeds  
**Constraints**: Reuse `SingleDebateScreen` and 006 endpoints; no duplicate vote semantics  
**Scale/Scope**: One new tab + one main screen + public feed route + user feed route + sqlc queries

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

**Code Quality Standards:**

- [x] TypeScript strict mode compliance verified (mobile app uses strict tsconfig)
- [x] ESLint — existing project config; new files must pass CI
- [x] Function complexity ≤ 10 — handlers split (feed builder vs HTTP)
- [x] Meaningful naming — `DebatesStack`, `MainDebatesScreen`, `getDebatesFeed`

**Testing Standards:**

- [x] TDD planned: Go tests for sqlc/feed query + handler; RN component tests for sections order (where feasible)
- [x] Integration: public feed (no auth) + user feed (auth middleware)
- [x] E2E: manual / Detox optional — P1 journey “tab → swipe → detail” documented in quickstart

**User Experience Consistency:**

- [x] Design system: dark theme + accent colors per mocks; loading/error on feed
- [x] Accessibility: min touch targets; screen reader labels on tabs and swipe actions
- [x] Auth gate on swipe when logged out (006)

**Performance Requirements:**

- [x] Feed capped with `new_limit` / `voted_limit` / public `limit` (**v1:** no cursor pagination); index review on `votes(debate_card_id, user_id)`
- [x] List `FlatList` with `keyExtractor` for main screen

**Developer Experience:**

- [x] OpenAPI fragment in `contracts/debates-feed.yaml`
- [x] [quickstart.md](./quickstart.md) for curl + Expo

**Post-design re-check**: Contracts and data model align with 006 card-vote completion rule — **PASS**.

## Project Structure

### Documentation (this feature)

```text
specs/009-debate-tab-navigator/
├── plan.md
├── research.md
├── spec.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── debates-feed.yaml
└── tasks.md              # created by /speckit.tasks — not this command
```

### Source Code (expected touchpoints)

```text
services/api/
├── sql/queries/debates.sql          # NEW: feed query(ies)
├── internal/api/debates.go          # NEW: getDebatesFeed handler
├── internal/api/api.go              # route registration
└── internal/database/               # sqlc regenerate

apps/mobile/
├── App.tsx                          # Tab navigator: add Debates tab + stack
├── src/screens/
│   ├── MainDebatesScreen.tsx        # NEW (or DebatesHomeScreen)
│   └── SingleDebateScreen.tsx       # REUSE navigate from activity list
├── src/services/                    # fetchDebatesFeed client
└── src/types/navigation.ts          # MainTabParamList + stack types
```

**Structure Decision**: Mobile + API (`apps/mobile` + `services/api`) — matches monorepo.

## Complexity Tracking

> No constitution violations requiring justification. Optional: single composite SQL for feed vs two queries — choose simpler maintainable query in implementation phase.

## Phase 0 & Phase 1 Outputs

| Artifact | Path |
|----------|------|
| Research | [research.md](./research.md) |
| Data model | [data-model.md](./data-model.md) |
| API contract | [contracts/debates-feed.yaml](./contracts/debates-feed.yaml) |
| Quickstart | [quickstart.md](./quickstart.md) |

## Next Steps

Run **`/speckit.tasks`** to generate `tasks.md` from this plan and implement in order: backend feed + sqlc → mobile tab + main screen → swipe UX → navigation to detail.
