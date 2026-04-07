# Implementation Plan: Internal Test Deployment Pipeline (iOS + Android)

**Branch**: `018-internal-test-deployment` | **Date**: 2026-04-06 | **Spec**: `specs/018-internal-test-deployment/spec.md`  
**Input**: Feature specification from `specs/018-internal-test-deployment/spec.md`

## Summary

Create a complete execution plan to deploy Fucci (Expo React Native) to Android and iOS internal test tracks with secure credentials, push setup (FCM/APNs via Expo), OTA channel isolation (`dev`/`staging`/`production`), and CI automation for build plus manually gated submit.

## Technical Context

**Language/Version**: TypeScript (Expo React Native), Go (backend integration touchpoints), GitHub Actions YAML  
**Primary Dependencies**: Expo CLI, EAS Build, EAS Submit, EAS Update, Google Play Console, Firebase FCM, Apple Developer, App Store Connect, GitHub Actions  
**Storage**: Supabase (Postgres + Realtime + Storage), GitHub/Expo secret stores for deployment credentials  
**Testing**: Existing repo test suites plus internal distribution smoke checks on both platforms  
**Target Platform**: Android internal testing (Play Internal Track), iOS internal testing (TestFlight Internal)  
**Project Type**: Monorepo mobile + API  
**Performance Goals**: Internal build turnaround < 30 minutes; deterministic credentialed builds; successful install and push verification on both platforms  
**Constraints**: Organization-owned release accounts only; CI auto-build + manual-gated submit; OTA channels isolated as `dev`, `staging`, `production`; secure secret handling  
**Scale/Scope**: One mobile app, 3 build profiles (`development`, `preview`, `production`), dual-platform internal distribution

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

**Code Quality Standards:**

- [x] TypeScript strict and lint requirements remain enforced for mobile config/workflow edits
- [x] Naming/structure rules are explicit in generated artifacts
- [x] No quality gate bypass introduced
- [x] Documentation remains maintainable and concise

**Testing Standards:**

- [x] Internal deployment smoke tests for core flows are defined
- [x] CI still requires pre-submit automated checks
- [x] Push and submission integration checks are explicitly required
- [x] Deterministic validation sequence is documented

**User Experience Consistency:**

- [x] Both platforms validate same core app journeys before release readiness
- [x] Failure and rollback checks included in runbook
- [x] Consistent update-channel behavior avoids cross-environment leakage
- [x] Async/loading/error handling verification included in smoke tasks

**Performance Requirements:**

- [x] Release pipeline timing targets documented
- [x] OTA and binary distribution separation reduces unnecessary rebuilds
- [x] No backend query-performance regression introduced by this scope
- [x] CI and runtime validation steps are measurable

**Developer Experience:**

- [x] Artifacts provide executable, dependency-aware guidance
- [x] API/workflow docs remain repository-relative
- [x] Setup automation via scripts preserved
- [x] Team ownership and secret custody policy documented

## Project Structure

### Documentation (this feature)

```text
specs/018-internal-test-deployment/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    └── internal-deployment.openapi.yaml
```

### Source Code (repository root)

```text
apps/mobile/
├── app.json
├── eas.json
└── package.json

.github/workflows/
└── deploy-api.yml

services/api/
└── (existing backend; deployment validation integration only)
```

**Structure Decision**: Keep current monorepo layout and focus plan outputs in `specs/018-internal-test-deployment/`, with actionable tasks targeting `apps/mobile/` and CI workflows.

## Complexity Tracking

No constitution violations identified for this planning scope.

## Post-Design Constitution Check

All constitution gates remain satisfied after design:

- Quality standards preserved.
- Test and validation gates are explicit and measurable.
- UX and performance verification are included for internal builds.
- Documentation and onboarding quality improved with dependency notes and clear ownership.
