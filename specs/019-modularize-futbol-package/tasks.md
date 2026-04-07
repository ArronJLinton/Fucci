# Tasks: Modularize Futbol Backend Package

**Input**: Design documents from `specs/019-modularize-futbol-package/`  
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`

**Tests**: Included because the spec explicitly requires unit and integration regression coverage.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (`[US1]`, `[US2]`, `[US3]`)
- Every task includes an explicit repository-relative file path

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish package skeleton and shared refactor scaffolding.

- [X] T001 Create new backend package directory and placeholder docs in `services/api/internal/futbol/README.md`
- [X] T002 Create package files `services/api/internal/futbol/{types.go,service.go,client.go,cache.go,transformer.go,summary.go}`
- [X] T003 [P] Add backend-only scope note to feature docs in `specs/019-modularize-futbol-package/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core abstractions needed before any story implementation.

**⚠️ CRITICAL**: No user story work starts before this phase is complete.

- [X] T004 Define canonical futbol domain DTOs and typed errors in `services/api/internal/futbol/types.go`
- [X] T005 Define `FutbolProvider` interface and `FutbolService` constructor/dependencies in `services/api/internal/futbol/service.go`
- [X] T006 Implement API-Football provider transport scaffolding (baseURL, headers, request helper wrappers) in `services/api/internal/futbol/client.go`
- [X] T007 Implement redesigned cache key namespace and TTL policy helpers (including stale-on-error semantics) in `services/api/internal/futbol/cache.go`
- [X] T008 Add mapper boundary helpers (provider payload -> canonical DTOs) in `services/api/internal/futbol/transformer.go`
- [X] T009 Wire new service instance into API config/wiring points in `services/api/internal/api/api.go`
- [X] T010 Create shared test fakes for provider/cache in `services/api/internal/futbol/test_helpers_test.go`

**Checkpoint**: `internal/futbol` foundational API exists and is injectable from handlers.

---

## Phase 3: User Story 1 - Stable API behavior after extraction (Priority: P1) 🎯 MVP

**Goal**: Preserve existing endpoint behavior while handlers become thin adapters.

**Independent Test**: Existing futbol endpoints return compatible status codes and response shapes for representative requests.

### Tests for User Story 1

- [X] T011 [P] [US1] Add contract regression tests for endpoint status/shape parity in `services/api/internal/api/futbol_contract_regression_test.go`
- [X] T012 [P] [US1] Add integration tests for route/query-name stability across `/futbol/*` endpoints in `services/api/internal/api/futbol_routes_integration_test.go`

### Implementation for User Story 1

- [X] T013 [US1] Refactor `getMatches` handler to delegate to `futbol.Service` in `services/api/internal/api/futbol.go`
- [X] T014 [US1] Refactor `getMatchLineup` handler to delegate to `futbol.Service` in `services/api/internal/api/futbol.go`
- [X] T015 [US1] Refactor `getLeagues` handler to delegate to `futbol.Service` in `services/api/internal/api/futbol.go`
- [X] T016 [US1] Refactor standings handlers (`getLeagueStandingsByTeamId`, `getLeagueStandingsByLeagueId`) to delegate to service in `services/api/internal/api/futbol.go`
- [X] T017 [US1] Implement service methods for matches/lineup/leagues/standings with unchanged HTTP-facing semantics in `services/api/internal/futbol/service.go`
- [X] T018 [US1] Keep endpoint parameter validation and error mapping at handler boundary in `services/api/internal/api/futbol.go`
- [X] T019 [US1] Update OpenAPI contract notes if needed for preserved compatibility in `specs/019-modularize-futbol-package/contracts/futbol-modularization.openapi.yaml`

**Checkpoint**: P1 routes behave compatibly with thin handlers and service-backed internals.

---

## Phase 4: User Story 2 - Extracted reusable futbol domain service (Priority: P2)

**Goal**: Move reusable logic out of handlers into testable package APIs.

**Independent Test**: `internal/futbol` methods are testable directly without HTTP handlers.

### Tests for User Story 2

- [X] T020 [P] [US2] Add unit tests for cache hit/miss + stale-on-error orchestration in `services/api/internal/futbol/service_cache_test.go`
- [X] T021 [P] [US2] Add unit tests for provider parse/error handling in `services/api/internal/futbol/client_test.go`
- [X] T022 [P] [US2] Add unit tests for transformer output invariants in `services/api/internal/futbol/transformer_test.go`

### Implementation for User Story 2

- [X] T023 [US2] Move reusable fetch operations (`FetchLineupData`, `FetchMatchStatsData`, standings/head-to-head helpers) into `services/api/internal/futbol/service.go`
- [X] T024 [US2] Move text/summary helper logic into `services/api/internal/futbol/summary.go`
- [X] T025 [US2] Move player/substitute normalization/filter helpers into `services/api/internal/futbol/transformer.go`
- [X] T026 [US2] Replace direct cache key string usage in handlers with package cache helpers from `services/api/internal/futbol/cache.go`
- [X] T027 [US2] Ensure debate aggregator consumes reusable service methods (not handler internals) in `services/api/internal/api/debate_data_aggregator.go`

**Checkpoint**: Core futbol logic is reusable from `internal/futbol` and independently unit-tested.

---

## Phase 5: User Story 3 - Pluggable provider abstraction (Priority: P3)

**Goal**: Enable future data-source swaps with minimal downstream changes.

**Independent Test**: Handlers/services run with mock provider implementation and no real network calls.

### Tests for User Story 3

- [X] T028 [P] [US3] Add mock-provider integration test for service behavior in `services/api/internal/futbol/service_provider_test.go`
- [X] T029 [P] [US3] Add handler wiring test with injected fake provider in `services/api/internal/api/futbol_provider_integration_test.go`

### Implementation for User Story 3

- [X] T030 [US3] Finalize `FutbolProvider` interface method set and API-Football implementation in `services/api/internal/futbol/{types.go,client.go}`
- [X] T031 [US3] Add provider injection/wiring path in API config constructors in `services/api/internal/api/api.go`
- [X] T032 [US3] Add provider-agnostic error mapping path in service/handler boundaries in `services/api/internal/futbol/service.go` and `services/api/internal/api/futbol.go`

**Checkpoint**: Provider can be swapped via wiring without endpoint contract rewrites.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize documentation, run full validation, and enforce backend-only scope.

- [X] T033 [P] Update refactor notes and validation results in `specs/019-modularize-futbol-package/quickstart.md`
- [X] T034 Run full backend test suite and capture results in `services/api/README.md`
- [X] T035 Confirm diff scope excludes client changes (`apps/mobile/**`, `apps/admin/**`) using `git diff --name-only`
- [X] T036 Run quickstart smoke checks and ensure contract parity summary in `specs/019-modularize-futbol-package/plan.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: no dependencies.
- **Phase 2 (Foundational)**: depends on Phase 1; blocks all user stories.
- **Phase 3 (US1/P1)**: depends on Phase 2; MVP slice.
- **Phase 4 (US2/P2)**: depends on Phase 3 service extraction baseline.
- **Phase 5 (US3/P3)**: depends on Phase 4 reusable abstractions.
- **Phase 6 (Polish)**: depends on all selected user stories complete.

### User Story Dependencies

- **US1 (P1)**: independent once foundational work is done.
- **US2 (P2)**: depends on US1 handler delegation baseline.
- **US3 (P3)**: depends on US2 service abstraction and tests.

### Parallel Opportunities

- Setup: `T003` can run with `T001`/`T002`.
- Foundational: `T006`, `T007`, `T008`, and `T010` can run in parallel after `T004`/`T005`.
- US1 tests `T011` and `T012` can run in parallel.
- US2 tests `T020`, `T021`, `T022` can run in parallel.
- US3 tests `T028`, `T029` can run in parallel.
- Polish: `T033` and `T035` can run in parallel.

---

## Parallel Example: User Story 1

```bash
# Run US1 regression tests in parallel:
Task: T011 services/api/internal/api/futbol_contract_regression_test.go
Task: T012 services/api/internal/api/futbol_routes_integration_test.go

# Then implement handler delegates in sequence:
Task: T013-T016 services/api/internal/api/futbol.go
```

## Parallel Example: User Story 2

```bash
# Build unit coverage in parallel:
Task: T020 services/api/internal/futbol/service_cache_test.go
Task: T021 services/api/internal/futbol/client_test.go
Task: T022 services/api/internal/futbol/transformer_test.go
```

## Implementation Strategy

### MVP First (US1 only)

1. Complete Phase 1 and Phase 2.
2. Deliver US1 (Phase 3) with contract regression tests green.
3. Validate no client-facing API regressions.

### Incremental Delivery

1. US1: thin handlers + compatibility parity.
2. US2: reusable package extraction + unit coverage.
3. US3: provider pluggability + injection tests.
4. Polish: backend-only scope verification + full test pass.

### Suggested MVP Scope

- **MVP**: Phase 1 + Phase 2 + Phase 3 (US1).
- This delivers safe refactor baseline with no client impact and preserved API behavior.
