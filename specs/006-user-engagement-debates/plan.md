# Implementation Plan: User Engagement for AI Powered Debates (006)

**Branch**: `006-user-engagement-debates` | **Date**: 2026-02-15 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/006-user-engagement-debates/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

006 adds: (1) headline + three AI-seeded comments (no stance labels in UI); (2) comment interactions (replies, upvote/downvote, emoji reactions) with auth gate for unauthenticated writes; (3) **swipe card voting** — users vote on the three debate cards via swipe right (yes) / swipe left (no) on a stacked card UI with thumbs up/down overlay, plus a **live debate meter** at the top and **team badge + score** in the header. Backend: Go (chi), PostgreSQL, existing `votes` (debate_card_id) for card votes; comment_votes/comment_reactions for comment engagement. Mobile: React Native (Expo), TypeScript; swipe gestures and card stack UI for voting.

## Technical Context

**Language/Version**: Go 1.22+, TypeScript (React Native / Expo)  
**Primary Dependencies**: chi (HTTP), React Navigation, expo; JWT auth (005)  
**Storage**: PostgreSQL (debates, debate_cards, comments, votes, comment_votes, comment_reactions); Redis for cache  
**Testing**: Go tests (internal/api, internal/ai); React Native / Jest for mobile  
**Target Platform**: Mobile (iOS/Android via Expo); API (Linux/server)  
**Project Type**: Monorepo — `apps/mobile`, `services/api`  
**Performance Goals**: API &lt; 200ms p95; mobile &lt; 3s load, &lt; 1s navigation; smooth 60fps swipe animations  
**Constraints**: One vote per user per debate_card (yes/no); auth required for card vote and comment writes  
**Scale/Scope**: Single debate screen with card stack (3 cards), comment list, live meter, header badge/score

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

**Code Quality Standards:**

- [x] TypeScript strict mode compliance verified (apps/mobile)
- [x] ESLint configuration defined with zero warnings
- [x] Function complexity ≤ 10, length ≤ 50 lines (target)
- [x] Meaningful naming conventions established

**Testing Standards:**

- [x] TDD approach planned for new features (API handlers, swipe flow)
- [x] Unit test coverage target ≥ 80% identified for business logic
- [x] Integration test requirements defined (API contract tests)
- [x] E2E test scenarios for P1 user stories planned (view debate, swipe vote, auth gate)

**User Experience Consistency:**

- [x] Design system compliance verified; loading/error states for comments and votes
- [x] Accessibility requirements (WCAG 2.1 AA) identified for swipe and focus
- [x] Loading states and error handling planned (comments, card vote submit, meter)
- [x] Responsive design considerations documented (mobile-first)

**Performance Requirements:**

- [x] Performance benchmarks defined (API &lt; 200ms p95, mobile &lt; 3s load)
- [x] Bundle size impact assessed (swipe/card stack components)
- [x] Database query performance targets set (votes aggregate for meter)
- [x] Caching strategy planned (debate + cards + vote counts)

**Developer Experience:**

- [x] Documentation requirements identified (quickstart, API contract)
- [x] API documentation needs defined (OpenAPI in contracts/)
- [x] Development environment setup documented (quickstart.md)
- [x] Code review guidelines established (constitution)

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
services/api/
├── internal/
│   ├── api/          # HTTP handlers (debates, comments, card votes)
│   ├── ai/           # Prompt generation
│   ├── auth/
│   ├── database/     # sqlc generated
│   └── cache/
├── sql/
│   ├── schema/       # Migrations (votes, comment_votes, comment_reactions)
│   └── queries/
└── cmd/

apps/mobile/
├── src/
│   ├── screens/      # SingleDebateScreen (card stack, meter, header, comments)
│   ├── components/   # Card stack, swipe overlay, debate meter, AuthGateModal
│   ├── services/     # api.ts (listComments, submitCardVote, etc.)
│   ├── navigation/
│   └── types/
└── __tests__/
```

**Structure Decision**: Monorepo with `services/api` (Go) and `apps/mobile` (React Native/Expo). Swipe card voting lives in mobile screens/components; card vote submission and vote aggregates in API. Existing debate/cards/comments/votes schema extended for card yes/no votes and meter aggregates.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation                  | Why Needed         | Simpler Alternative Rejected Because |
| -------------------------- | ------------------ | ------------------------------------ |
| [e.g., 4th project]        | [current need]     | [why 3 projects insufficient]        |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient]  |
