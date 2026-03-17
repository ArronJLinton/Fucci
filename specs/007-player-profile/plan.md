# Implementation Plan: 007 – Player Profile

**Branch**: `007-player-profile` | **Date**: 2026-02-15 | **Spec**: `specs/007-player-profile/spec.md`
**Input**: Feature specification from `/specs/007-player-profile/spec.md`

**Note**: Generated via `/speckit.plan` for the Player Profile user experience.

## Summary

Implement a FIFA-style Player Profile experience in the mobile app, allowing authenticated users to create a single player profile with core attributes (age, country, club/free agent, position), traits, profile photo, and career teams.  
The feature will extend the existing React Native/Expo client and Go/PostgreSQL API with new player profile endpoints, a simple traits enum, and S3-backed photo storage, following Fucci’s constitution for testing, performance, and UX.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: React Native (TypeScript) + Expo; Go 1.22 API; PostgreSQL 15  
**Primary Dependencies**: React Navigation, React Query (if used), React Native Image Picker, existing Go HTTP stack, S3-compatible object storage  
**Storage**: PostgreSQL tables for player profiles, traits, and career teams; S3 (or equivalent) for profile photos  
**Testing**: Jest + React Native Testing Library for mobile; Go `testing` + testify for API handlers; integration tests for profile endpoints  
**Target Platform**: iOS and Android mobile apps (Expo), existing Go API service
**Project Type**: Mobile + API (monorepo: `apps/mobile`, `services/api`)  
**Performance Goals**: 60 fps UI; profile screen interactions < 500ms; profile API p95 latency < 200ms  
**Constraints**: Respect existing mobile bundle size limits; profile photo uploads ≤ 5 MB; API endpoints must follow auth + security patterns already in the service  
**Scale/Scope**: Initially thousands of users; one profile per user; 1–2 screens plus 1–2 modals in the mobile app

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

**Code Quality Standards:**

- [x] TypeScript strict mode compliance verified for new mobile code (existing config enforced)
- [x] ESLint configuration with zero new warnings for this feature
- [x] Function complexity ≤ 10, length ≤ 50 lines (justify exceptions in PR if needed)
- [x] Meaningful naming conventions aligned with existing debate/profile modules

**Testing Standards:**

- [x] TDD approach planned for new profile endpoints and UI components
- [x] Unit test coverage target ≥ 80% for new Player Profile logic (API + mobile)
- [x] Integration test requirements defined for profile CRUD, traits, and career teams
- [x] E2E test scenarios planned for create/edit profile happy path and photo upload failure

**User Experience Consistency:**

- [x] Design system compliance with existing Fucci typography, spacing, and button styles
- [x] Accessibility requirements (WCAG 2.1 AA) identified: labels, touch targets, contrast
- [x] Loading states and error handling planned for profile load/save and photo upload
- [x] Responsive design considerations documented for small/large phone sizes

**Performance Requirements:**

- [x] Performance benchmarks defined (profile screen load < 500ms after API, interactions < 200ms)
- [x] Bundle size impact assessed (image picker + new components within mobile limits)
- [x] Database query performance targets set (profile queries p95 < 100ms)
- [x] Caching strategy planned (mobile query caching where appropriate; API-level DB indices)

**Developer Experience:**

- [x] Documentation requirements identified (quickstart, data-model, contracts in `specs/007-player-profile`)
- [x] API documentation needs defined (OpenAPI contracts for profile endpoints)
- [x] Development environment setup documented in quickstart.md for this feature
- [x] Code review guidelines established by referencing Fucci constitution

## Project Structure

### Documentation (this feature)

```text
specs/007-player-profile/
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
│   ├── api/             # HTTP handlers (new player_profile handlers live here)
│   ├── database/        # sqlc-generated queries for player_profiles, traits, career_teams
│   └── cache/           # shared caching abstractions (if used)
└── sql/
    └── queries/         # SQL for player profile entities

apps/mobile/
└── src/
    ├── screens/
    │   └── PlayerProfileScreen.tsx
    ├── components/
    │   └── PlayerTraitsModal.tsx
    └── services/
        └── playerProfile.ts   # client API for profile CRUD, traits, career teams
```

**Structure Decision**: Mobile + API within the existing monorepo (`apps/mobile` + `services/api`), with one main screen and supporting components on mobile and a small set of focused handlers + queries on the API side.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation                  | Why Needed         | Simpler Alternative Rejected Because |
| -------------------------- | ------------------ | ------------------------------------ |
| [e.g., 4th project]        | [current need]     | [why 3 projects insufficient]        |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient]  |
