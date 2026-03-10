# Implementation Plan: User Registration and Settings Flow

**Branch**: `005-user-registration-settings` | **Date**: 2026-03-09 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/005-user-registration-settings/spec.md`

## Summary

Deliver mobile UI and aligned API for **Sign up** (username/email, password, first name, last name, optional photo), **Login** (email/username + password, Remember me, Forgot password), and **Settings** with tabs: **Following**, **Player Profile**, **Team Manager**, and **Logout**. Backend auth and user APIs exist in 001; this feature extends the registration payload and profile shape (first_name, last_name, photo) and defines the settings screen structure and navigation.

## Technical Context

**Language/Version**: TypeScript (React Native/Expo), Go 1.22+ (backend)  
**Primary Dependencies**: React Navigation, existing auth API (JWT), 001 user/follow APIs  
**Storage**: PostgreSQL (users, user_follows); existing schema in 001; may add first_name, last_name, avatar_url if not present  
**Testing**: Jest / React Native Testing Library (mobile), Go tests (API); E2E for sign-up and login flows  
**Target Platform**: iOS/Android via Expo; API on existing Go service  
**Project Type**: Mobile app + API (apps/mobile, services/api)  
**Performance Goals**: Sign-up/Login < 2s; Settings screen < 1s; API p95 < 200ms  
**Constraints**: WCAG 2.1 AA, design system (dark theme, blue accents from FUCCI flows)  
**Scale/Scope**: Single settings screen, 3 tabs + Logout; reuse existing auth and /users/me

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

**Code Quality Standards:**

- [x] TypeScript strict mode compliance verified (mobile)
- [x] ESLint configuration defined with zero warnings
- [x] Function complexity ≤ 10, length ≤ 50 lines (target)
- [x] Meaningful naming conventions established

**Testing Standards:**

- [x] TDD approach planned for new features
- [x] Unit test coverage target ≥ 80% identified for auth/settings logic
- [x] Integration test requirements defined (auth API, /users/me)
- [x] E2E test scenarios for P1 user stories planned (sign-up, login)

**User Experience Consistency:**

- [x] Design system compliance verified (FUCCI flows, dark theme)
- [x] Accessibility requirements (WCAG 2.1 AA) identified
- [x] Loading states and error handling planned (forms, API errors)
- [x] Responsive design considerations documented (mobile-first)

**Performance Requirements:**

- [x] Performance benchmarks defined (load times, latency)
- [x] Bundle size impact assessed (minimal; new screens only)
- [x] Database query performance targets set (reuse existing indexes)
- [x] Caching strategy planned (token/session; no new cache for MVP)

**Developer Experience:**

- [x] Documentation requirements identified (quickstart, API contract)
- [x] API documentation needs defined (contracts in this spec)
- [x] Development environment setup documented (existing quickstart)
- [x] Code review guidelines established (constitution)

## Project Structure

### Documentation (this feature)

```text
specs/005-user-registration-settings/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (API contract)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
apps/mobile/
├── src/
│   ├── screens/
│   │   ├── SignUpScreen.tsx      # New: registration form
│   │   ├── LoginScreen.tsx       # New or existing: login form
│   │   ├── SettingsScreen.tsx    # New: tabs Following, Player Profile, Team Manager, Logout
│   │   └── ...
│   ├── navigation/              # Register new screens; auth stack vs main
│   └── services/
│       └── api.ts               # Extend with register/login/settings API calls
services/api/
├── internal/
│   └── api/
│       └── auth.go / users.go    # Extend register to first_name, last_name, photo; profile GET/PUT
└── sql/                         # Migrations if user table extended
```

**Structure Decision**: Monorepo; mobile app in apps/mobile, API in services/api. New screens and navigation entries for Sign Up, Login, Settings; API extended to accept first_name, last_name, optional photo on register and on profile update.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
| --------- | ---------- | ------------------------------------ |
| None | — | — |
