# Implementation Plan: Debate Tab & Main Debates Experience

**Branch**: `009-debate-tab-navigator` | **Date**: 2026-03-28 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/009-debate-tab-navigator/spec.md`  
**Plan refresh**: `/speckit.plan` — main debates are **sourced from top world football news and headlines** (see Summary + [research.md](./research.md) §7).

## Summary

Add a **Debates** tab to the Expo bottom navigator with a **main debates** screen: **new** (incomplete card-vote) debates on top, **voted** debates below; **Tinder-style** swipes on the featured card (**right = agree/upvote**, **left = disagree/downvote**) wired to existing card-vote APIs; tap **My Activity** rows **or** (for **guests**) a public browse row to open **debate detail** (`SingleDebate`) — **guests** read headline, meter, and comments; **signed-in** users get full engagement per **006**.

**Debate content sourcing (product)**: Debates shown in the main feed should be **grounded in current world football (soccer) news and headlines**. Implementation does **not** add a separate news crawler in 009. Instead, **generation and prioritization** follow [004-ai-debate-generator](../004-ai-debate-generator/spec.md): the AI debate generator’s **context bundle** already requires **news articles** as an input; jobs/on-demand generation should emphasize **top/trending football headlines** so the Debates tab reflects real-world narrative. The **009** work exposes those debates via **public** + **authenticated** feed APIs; any new DB fields for headline provenance are **optional** and may land in 004 or a small migration (see [data-model.md](./data-model.md)).

Backend adds:

- **`GET /v1/api/debates/public-feed`** — **no auth**, read-only debate summaries for **guests** (browse).
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
**Constraints**: Reuse `SingleDebateScreen` and 006 endpoints; no duplicate vote semantics; **009 does not own news ingestion** — ties to headlines via **004** generator and existing news APIs (`/v1/api/news/...` as applicable)  
**Scale/Scope**: One new tab + one main screen + public feed route + user feed route + sqlc queries; **news → debate** narrative owned by **004** jobs/context building

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

**Post-design re-check**: Contracts and data model align with 006 card-vote completion rule; news-headline provenance documented as **004-aligned** optional DTO fields — **PASS**.

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
└── tasks.md
```

### Source Code (expected touchpoints)

```text
services/api/
├── sql/queries/debates.sql          # feed query(ies) + public ordering
├── internal/api/debates.go          # public-feed + user feed handlers
├── internal/api/api.go              # route registration
└── internal/database/               # sqlc regenerate

apps/mobile/
├── App.tsx                          # Tab navigator: Debates tab + stack
├── src/screens/
│   ├── MainDebatesScreen.tsx
│   └── SingleDebateScreen.tsx       # REUSE
├── src/services/debate.ts
└── src/types/navigation.ts
```

**Structure Decision**: Mobile + API (`apps/mobile` + `services/api`) — matches monorepo. **News → headline context** for new debates: extend or configure **004** generation/workers (and existing news routes), not new scrapers inside 009.

## Complexity Tracking

> No constitution violations requiring justification.

## Phase 0 & Phase 1 Outputs

| Artifact | Path |
|----------|------|
| Research | [research.md](./research.md) |
| Data model | [data-model.md](./data-model.md) |
| API contract | [contracts/debates-feed.yaml](./contracts/debates-feed.yaml) |
| Quickstart | [quickstart.md](./quickstart.md) |

## Next Steps

Implement per [tasks.md](./tasks.md): backend feeds + sqlc → mobile tab + main screen → swipe → navigation to detail. Coordinate with **004** if adjusting generation prompts to weight **world football** headlines for debates appearing in the main tab.
