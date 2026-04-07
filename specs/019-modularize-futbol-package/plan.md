# Implementation Plan: Modularize Futbol Backend Package

**Branch**: `019-modularize-futbol-package` | **Date**: 2026-04-07 | **Spec**: `specs/019-modularize-futbol-package/spec.md`
**Input**: Feature specification from `specs/019-modularize-futbol-package/spec.md`

## Summary

Refactor `services/api/internal/api/futbol.go` into a dedicated `services/api/internal/futbol/` domain package (client/service/cache/transform/types) while preserving the existing public API surface (status codes, field names/types, route/query naming). This feature is backend-only and explicitly excludes client/mobile/web changes.

## Technical Context

**Language/Version**: Go 1.24 toolchain (`services/api/go.mod`)  
**Primary Dependencies**: Go stdlib, existing internal cache interface (`services/api/internal/cache`), existing API HTTP helpers in `services/api/internal/api`  
**Storage**: N/A (no schema/storage changes in this feature)  
**Testing**: `go test` (existing handler tests + new `internal/futbol` unit tests)  
**Target Platform**: Linux container backend API on Fly  
**Project Type**: Monorepo with backend service in `services/api`  
**Performance Goals**: Maintain current endpoint latency/cache-hit behavior; stale-on-error fallback on provider failures  
**Constraints**: Backend-only scope; no client/mobile/web file changes; preserve endpoint contract compatibility and route/query names  
**Scale/Scope**: Extract and modularize ~900 LOC from one backend handler file into reusable package boundaries

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

**Code Quality Standards:**

- [x] Backend package split enforces single responsibility (`handler` vs `service/client/cache/transform`)
- [x] Exported package APIs and types must have meaningful names and concise docs
- [x] Function size/complexity reduced from monolithic handler file via modular extraction
- [x] TypeScript/ESLint checks are N/A (no frontend/client scope)

**Testing Standards:**

- [x] Existing endpoint tests serve as regression baseline for no-contract-break refactor
- [x] Unit tests planned for new `internal/futbol` package with >=80% target on extracted logic
- [x] Integration tests defined for handler-to-service wiring and stale-on-error behavior
- [x] E2E/user-journey tests are out-of-scope (backend-only refactor, no client journey changes)

**User Experience Consistency:**

- [x] No UI surface changes; client behavior remains stable via API contract preservation
- [x] Existing error response shape semantics preserved for client compatibility
- [x] Accessibility/responsive requirements are unchanged (no client modifications)
- [x] Backend stale-on-error behavior maintains predictable user-facing outcomes

**Performance Requirements:**

- [x] Backend latency/caching behavior maintained or improved
- [x] Bundle size impact is N/A (no client changes)
- [x] Database query targets are N/A (no DB logic changes)
- [x] Cache strategy redesign is explicitly in scope and test-validated

**Developer Experience:**

- [x] Feature docs generated under `specs/019-modularize-futbol-package/`
- [x] Contract artifact documents preserved endpoint surface
- [x] Quickstart includes backend-only validation commands
- [x] Repository-relative paths used throughout artifacts

## Project Structure

### Documentation (this feature)

```text
specs/019-modularize-futbol-package/
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
services/api/internal/
├── api/
│   ├── futbol.go                    # HTTP handlers (thin post-refactor)
│   ├── debate_data_aggregator.go    # consumes reusable futbol service methods
│   └── types.go
├── futbol/                          # new backend package
│   ├── client.go
│   ├── service.go
│   ├── cache.go
│   ├── transformer.go
│   ├── summary.go
│   └── types.go
├── news/                            # pattern reference
└── cache/

services/api/internal/api/*_test.go
services/api/internal/futbol/*_test.go

apps/mobile/**                       # unchanged (out of scope)
```

**Structure Decision**: Backend-only refactor using `internal/news` as architectural template; no client-layer files change.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
| --------- | ---------- | ------------------------------------ |
| N/A       | N/A        | N/A                                  |

## Contract Parity Summary (Post-Implementation)

- Route and query parameter names preserved for:
  - `/futbol/matches?date=...&league_id=...`
  - `/futbol/lineup?match_id=...`
  - `/futbol/leagues`
  - `/futbol/team_standings?team_id=...`
  - `/futbol/league_standings?league_id=...&season=...`
- Error-shape and route stability regression tests are in place and passing in `internal/api`.
- Provider abstraction is injectable (`Config.FutbolProvider`) with service-level typed error normalization.
- Backend-only scope confirmed: `git diff --name-only main...HEAD -- apps/mobile apps/admin` returned no files.
