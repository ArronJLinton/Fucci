# Tasks: Internal Test Deployment Pipeline (iOS + Android)

**Input**: Design documents from `specs/018-internal-test-deployment/`  
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/internal-deployment.openapi.yaml`, `quickstart.md`

**Tests**: No net-new unit/integration test files are required by the spec; validation is done through internal build, submission, push, and OTA smoke checks.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated independently.

## Format: `[ID] [P?] [Story] Description`

- [X] T001 Create deployment execution notes scaffold in `specs/018-internal-test-deployment/quickstart.md`
- [X] T002 Capture account ownership and access matrix in `specs/018-internal-test-deployment/spec.md`
- [X] T003 [P] Add CI secret inventory section to `specs/018-internal-test-deployment/quickstart.md`
- [X] T004 [P] Add credential asset checklist details to `specs/018-internal-test-deployment/data-model.md`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish deployment documentation and execution scaffolding.

- [X] T005 Normalize EAS/Expo command examples and prerequisites in `specs/018-internal-test-deployment/quickstart.md`
- [X] T006 [P] Add operator run commands for local preflight checks in `apps/mobile/package.json`
- [X] T007 [P] Add environment variable reference table for mobile deploy jobs in `specs/018-internal-test-deployment/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core configuration baseline required before any user story can be completed.

**⚠️ CRITICAL**: No user story work should be marked complete until this phase is complete.

- [X] T008 Verify and document Android/iOS identifiers and ownership constraints in `apps/mobile/app.json`
- [X] T009 Create and document `development`, `preview`, `production` profile baseline in `apps/mobile/eas.json`
- [X] T010 [P] Add OTA channel mapping (`dev`, `staging`, `production`) to `apps/mobile/eas.json`
- [X] T011 [P] Add runtimeVersion policy and release-channel notes in `specs/018-internal-test-deployment/quickstart.md`
- [X] T012 Add credential backup and custody procedure in `specs/018-internal-test-deployment/quickstart.md`

**Checkpoint**: Foundation ready for independent user-story execution.

---

## Phase 3: User Story 1 - Internal Store/Test Track Readiness (Priority: P1) 🎯 MVP

**Goal**: Make both app stores and push providers ready for internal testing distribution.

**Independent Test**: Play Internal Track and TestFlight are configured with app records/tester groups, and push credentials are documented and ready for EAS.

### Implementation for User Story 1

- [X] T013 [US1] Document Google Play Console setup tasks and dependency blockers in `specs/018-internal-test-deployment/quickstart.md`
- [X] T014 [US1] Document Firebase project + FCM credential setup steps in `specs/018-internal-test-deployment/quickstart.md`
- [X] T015 [US1] Document Apple Developer enrollment and App ID creation steps in `specs/018-internal-test-deployment/quickstart.md`
- [X] T016 [US1] Document APNs key creation and secure storage process in `specs/018-internal-test-deployment/quickstart.md`
- [X] T017 [US1] Document App Store Connect app record and TestFlight internal group setup in `specs/018-internal-test-deployment/quickstart.md`
- [X] T018 [P] [US1] Add push environment readiness fields and acceptance criteria in `specs/018-internal-test-deployment/data-model.md`
- [X] T019 [US1] Add US1 verification checklist (records, testers, push keys present) in `specs/018-internal-test-deployment/quickstart.md`

**Checkpoint**: Platform accounts, app records, and push prerequisites are fully defined and testable.

---

## Phase 4: User Story 2 - EAS Build, Submit, and OTA Configuration (Priority: P2)

**Goal**: Ensure Expo/EAS config supports secure internal builds and environment-isolated OTA updates.

**Independent Test**: `eas build` can be run for both platforms with preview profile; OTA updates target `staging` only for preview builds.

### Implementation for User Story 2

- [X] T020 [US2] Configure EAS submit profile defaults for Android internal and iOS TestFlight in `apps/mobile/eas.json`
- [X] T021 [US2] Add profile-to-channel mapping notes (`preview -> staging`) in `specs/018-internal-test-deployment/quickstart.md`
- [X] T022 [US2] Add EAS credential lifecycle tasks (keystore, provisioning, APNs upload) in `specs/018-internal-test-deployment/quickstart.md`
- [X] T023 [P] [US2] Add release-channel runtime compatibility checks in `specs/018-internal-test-deployment/quickstart.md`
- [X] T024 [P] [US2] Add OTA publish/rollback command runbook for `staging` in `specs/018-internal-test-deployment/quickstart.md`
- [X] T025 [US2] Update deployment API contract examples for profile/channel metadata in `specs/018-internal-test-deployment/contracts/internal-deployment.openapi.yaml`
- [X] T026 [US2] Add US2 validation checklist for EAS build/submit/update readiness in `specs/018-internal-test-deployment/quickstart.md`

**Checkpoint**: EAS and OTA configuration is complete and independently verifiable.

---

## Phase 5: User Story 3 - Automated CI Build + Gated Submission (Priority: P3)

**Goal**: Automate mobile internal builds and enforce manual approval before store submissions.

**Independent Test**: GitHub Actions builds both platforms automatically and only allows submit steps through explicit manual approval.

### Implementation for User Story 3

- [X] T027 [US3] Create mobile deployment workflow with Android/iOS preview build jobs in `.github/workflows/mobile-internal-deploy.yml`
- [X] T028 [US3] Add gated submit jobs using protected environment/manual dispatch in `.github/workflows/mobile-internal-deploy.yml`
- [X] T029 [P] [US3] Add workflow secret validation and fail-fast checks in `.github/workflows/mobile-internal-deploy.yml`
- [X] T030 [P] [US3] Add artifact/log retention and notification steps in `.github/workflows/mobile-internal-deploy.yml`
- [X] T031 [US3] Document required GitHub secrets and setup commands in `specs/018-internal-test-deployment/quickstart.md`
- [X] T032 [US3] Add CI job graph and dependency notes aligned with workflow jobs in `specs/018-internal-test-deployment/quickstart.md`
- [X] T033 [US3] Add US3 run validation checklist (build complete, approval gate, submit success) in `specs/018-internal-test-deployment/quickstart.md`

**Checkpoint**: CI pipeline is operational and submissions are safely gated.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final hardening and end-to-end readiness confirmation across stories.

- [ ] T034 [P] Reconcile terminology (`preview` profile vs `staging` OTA channel) across `specs/018-internal-test-deployment/spec.md`
- [ ] T035 [P] Reconcile terminology (`preview` profile vs `staging` OTA channel) across `specs/018-internal-test-deployment/plan.md`
- [ ] T036 Consolidate final dependency-ordered execution list in `specs/018-internal-test-deployment/quickstart.md`
- [ ] T037 Execute and record Android and iOS internal smoke validation results in `specs/018-internal-test-deployment/quickstart.md`
- [ ] T038 Execute and record push + OTA staging validation and rollback evidence in `specs/018-internal-test-deployment/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- Setup (Phase 1) -> Foundational (Phase 2) -> User Stories (Phase 3-5) -> Polish (Phase 6)
- User stories begin only after Phase 2 is complete.

### User Story Dependencies

- **US1 (P1)**: Depends on Phase 2; no dependency on US2/US3.
- **US2 (P2)**: Depends on Phase 2 and outputs from US1 account/push readiness.
- **US3 (P3)**: Depends on Phase 2 and uses finalized EAS settings from US2.

### Within Each User Story

- Account/credential setup before verification checklist tasks.
- EAS profile/channel definitions before build/submit/OTA validation tasks.
- CI workflow creation before workflow documentation and execution checks.

### Parallel Opportunities

- Setup: T006 and T007 can run in parallel.
- Foundational: T010 and T011 can run in parallel.
- US1: T018 can run in parallel with T013-T017.
- US2: T023 and T024 can run in parallel.
- US3: T029 and T030 can run in parallel.
- Polish: T034 and T035 can run in parallel.

---

## Parallel Example: User Story 3

```bash
Task: "Add workflow secret validation and fail-fast checks in .github/workflows/mobile-internal-deploy.yml"
Task: "Add artifact/log retention and notification steps in .github/workflows/mobile-internal-deploy.yml"
```

---

## Implementation Strategy

### MVP First (US1)

1. Complete Phase 1 and Phase 2.
2. Deliver US1 (store/test-track + push prerequisite readiness).
3. Validate US1 checklist before moving to automation.

### Incremental Delivery

1. US1: Accounts, app records, push prerequisites.
2. US2: EAS build/submit/OTA config stabilization.
3. US3: CI automation and manual-gated submission.
4. Phase 6: End-to-end smoke evidence and final sign-off.

### Parallel Team Strategy

1. Engineer A: US1 platform account + push setup docs.
2. Engineer B: US2 EAS/OTA config and runbook.
3. Engineer C: US3 CI workflow and secret gating.

---

## Notes

- All tasks follow the required checklist format with IDs and exact file paths.
- `[P]` indicates parallelizable tasks.
- `[US1]`, `[US2]`, and `[US3]` labels map directly to independent delivery slices.
