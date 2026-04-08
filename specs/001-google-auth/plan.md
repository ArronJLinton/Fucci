# Implementation Plan: Google OAuth Registration & Sign-In

**Branch**: `001-google-auth` | **Date**: 2026-04-08 | **Spec**: `specs/001-google-auth/spec.md`  
**Input**: Feature specification from `specs/001-google-auth/spec.md`

## Summary

Implement one backend endpoint, `POST /auth/google`, that handles both registration and sign-in using Google OAuth 2.0 authorization codes. Verify Google-issued identity data, enforce `email_verified`, prevent password-account collisions, issue Fucci JWTs, and return `is_new` for mobile routing. Add required user schema changes (`google_id`, `auth_provider`, `avatar_url`, `locale`, `last_login_at`) and wire mobile Sign Up/Login to launch Google auth and exchange the code with backend.

## Technical Context

**Language/Version**: Go 1.22+ (API), TypeScript strict mode (React Native mobile)  
**Primary Dependencies**: Gin HTTP server, JWT issuance utilities, Google token exchange/verification library (`google-auth-library` per requirement), Expo/React Native auth session integrations  
**Storage**: PostgreSQL `users` table migration updates  
**Testing**: Go unit + integration tests, mobile unit tests, E2E flow validation for US-01/US-02  
**Target Platform**: API service + iOS/Android mobile clients  
**Project Type**: Monorepo mobile + API  
**Performance Goals**: Auth endpoint p95 < 200ms excluding external Google latency; mobile auth completion in < 10s for successful flows  
**Constraints**: Must return specified error codes; no account linking in this feature; preserve existing email/password auth behavior  
**Scale/Scope**: Registration and login entry points on mobile + single backend auth endpoint + one user-table migration set

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

### Pre-Design Gate

**Code Quality Standards:**

- [x] TypeScript strict mode compliance verified
- [x] ESLint configuration defined with zero warnings
- [x] Function complexity ≤ 10, length ≤ 50 lines
- [x] Meaningful naming conventions established

**Testing Standards:**

- [x] TDD approach planned for new features
- [x] Unit test coverage target ≥ 80% identified
- [x] Integration test requirements defined
- [x] E2E test scenarios for P1 user stories planned

**User Experience Consistency:**

- [x] Design system compliance verified
- [x] Accessibility requirements (WCAG 2.1 AA) identified
- [x] Loading states and error handling planned
- [x] Responsive design considerations documented

**Performance Requirements:**

- [x] Performance benchmarks defined (load times, latency)
- [x] Bundle size impact assessed
- [x] Database query performance targets set
- [x] Caching strategy planned

**Developer Experience:**

- [x] Documentation requirements identified
- [x] API documentation needs defined
- [x] Development environment setup documented
- [x] Code review guidelines established

### Post-Design Gate Re-check

- [x] No unresolved clarifications remain after research
- [x] Data model and contracts align with user scenarios and constraints
- [x] Planned tests cover unit, integration, and P1 E2E journeys
- [x] Paths in docs are repository-relative and portable

## Project Structure

### Documentation (this feature)

```text
specs/001-google-auth/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── auth-google.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
services/api/
├── internal/
│   ├── api/
│   │   ├── auth.go
│   │   └── auth_google_test.go
│   ├── db/
│   └── middleware/
└── sql/
    └── migrations/

apps/mobile/src/
├── screens/
│   ├── auth/
│   │   ├── LoginScreen.tsx
│   │   └── SignUpScreen.tsx
├── services/
│   └── auth.ts
├── navigation/
└── __tests__/
```

**Structure Decision**: Use existing API + mobile app structure with focused additions in auth handlers/services, SQL migrations, and auth screens/services; avoid introducing new top-level modules.

## Phase 0 Research Output

Research decisions are captured in `specs/001-google-auth/research.md` and resolve provider-flow, token verification, account-collision handling, and platform redirect behavior for iOS/Android.

## Phase 1 Design Output

- Data model: `specs/001-google-auth/data-model.md`
- Contracts: `specs/001-google-auth/contracts/auth-google.openapi.yaml`
- Validation quickstart: `specs/001-google-auth/quickstart.md`
- Agent context updated via `.specify/scripts/bash/update-agent-context.sh cursor-agent`

## Complexity Tracking

No constitution violations requiring justification.
