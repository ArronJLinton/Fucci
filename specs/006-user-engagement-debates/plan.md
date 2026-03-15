# Implementation Plan: User Engagement for AI Powered Debates

**Branch**: `006-user-engagement-debates` | **Date**: 2026-02-15 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/006-user-engagement-debates/spec.md`

## Summary

Three enhancements to the AI Powered Debates experience: (1) **Debate structure**: headline + description + three AI-seeded comments (agree, disagree, wildcard) attributed to the **system user** (Fucci), stored as Comment rows with `seeded: true`; no stance labels — rendered and stored as regular comments; create system user if one does not exist; Fucci is the attributed author for moderation/liability. (2) **Comment interactions**: replies and subcomments (one level), comment-level upvotes/downvotes, emoji reactions (no max on emoji types per comment); unauthenticated users can **see** vote and reaction counts but cannot engage; all write actions require auth. (3) **Auth gate modal** (mobile only; web not available): when an unauthenticated user attempts reply, vote, or reaction, show a modal; after auth, deep-link back to the debate and auto-initiate the blocked action where feasible. Extends 004 (debates, generation) and 005 (auth, login/register).

## Technical Context

**Language/Version**: TypeScript (React Native/Expo), Go 1.22+ (backend)  
**Primary Dependencies**: React Navigation, existing debate API (004), auth API (005), JWT  
**Storage**: PostgreSQL; extend `comments` (add `seeded`); new tables `comment_votes`, `comment_reactions`  
**Testing**: Jest / React Native Testing Library (mobile), Go tests (API); E2E for comment reply, vote, reaction and auth gate  
**Target Platform**: iOS/Android via Expo (mobile only; web not available); API on existing Go service (chi)  
**Project Type**: Mobile app + API (apps/mobile, services/api)  
**Performance Goals**: List comments < 200ms p95; vote/reaction toggle < 100ms; auth gate modal instant  
**Constraints**: One-level subcomments only; no max on emoji reaction types per comment; seeded flag and stance not exposed in UI; return-to-debate deep link on mobile only  
**Scale/Scope**: Per-debate comments (hundreds per debate); comment_votes and comment_reactions indexed by comment_id; system user (Fucci) for all seeded comments

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

**Code Quality Standards:**

- [x] TypeScript strict mode compliance verified (mobile)
- [x] ESLint configuration defined with zero warnings
- [x] Function complexity ≤ 10, length ≤ 50 lines (target)
- [x] Meaningful naming conventions established

**Testing Standards:**

- [x] TDD approach planned for new features (comment API, vote/reaction, auth gate)
- [x] Unit test coverage target ≥ 80% for comment and vote/reaction logic
- [x] Integration test requirements defined (comments CRUD, vote, reaction endpoints)
- [x] E2E test scenarios for P1 user stories planned (reply, vote, react, auth gate return)

**User Experience Consistency:**

- [x] Design system compliance (debate UI, modal, buttons)
- [x] Accessibility requirements (WCAG 2.1 AA) for modal and comment actions
- [x] Loading states and error handling for comment/vote/reaction API calls
- [x] Responsive design (mobile-first; modal and lists)

**Performance Requirements:**

- [x] Performance benchmarks defined (comment list, vote/reaction latency)
- [x] Bundle size impact assessed (new screens/components minimal)
- [x] Database query performance (indexes on comment_id for votes/reactions; comment list by debate_id)
- [x] Caching strategy (optional: cache comment list per debate with invalidation on write)

**Developer Experience:**

- [x] Documentation requirements (quickstart, API contract in contracts/api.yaml)
- [x] API documentation (OpenAPI for comments, vote, reaction)
- [x] Development environment setup (existing quickstart; migrations for new tables)
- [x] Code review guidelines (constitution)

## Project Structure

### Documentation (this feature)

```text
specs/006-user-engagement-debates/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (API contract)
│   └── api.yaml
└── tasks.md             # Phase 2 output (/speckit.tasks — not created by /speckit.plan)
```

### Source Code (repository root)

```text
apps/mobile/
├── src/
│   ├── screens/
│   │   ├── SingleDebateScreen.tsx   # Extend: comments list, reply, vote, reaction UI; auth gate modal
│   │   └── ...
│   ├── components/                 # Optional: CommentCard, VoteButtons, ReactionPicker, AuthGateModal
│   ├── context/
│   │   └── AuthContext.tsx         # Use for auth gate (005)
│   └── services/
│       └── api.ts                  # Add: listComments, createComment, setCommentVote, addCommentReaction

services/api/
├── internal/
│   └── api/
│       ├── debates.go              # Extend: create seeded comments when generating debate; optional
│       ├── comments.go             # New: list, create comment; enforce subcomment depth
│       └── comment_engagement.go   # New or part of comments: vote, reaction handlers
├── internal/
│   └── database/                   # sqlc: comment_votes, comment_reactions queries
└── sql/
    └── schema/                     # Migrations: comments.seeded; comment_votes; comment_reactions
```

**Structure Decision**: Monorepo; mobile app in apps/mobile (extend SingleDebateScreen and API client), API in services/api (new comment and comment-engagement handlers, new tables). Auth gate is client-only (modal + navigation to login/register from 005).

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | — | — |
