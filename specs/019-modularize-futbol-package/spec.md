# Feature Specification: Modularize Futbol Backend Package

**Feature Branch**: `019-modularize-futbol-package`  
**Created**: 2026-04-07  
**Status**: Draft  
**Input**: User description: "We need to modularize the `services/api/internal/api/futbol.go` logic, operations, and handlers into its own package, similar to `services/api/internal/news/`. The objective is to increase reusability, maintainability, and make it easier to change the source of futbol (soccer) data."

## Clarifications

### Session 2026-04-07

- Q: What compatibility level should be required during refactor for existing futbol endpoints? → A: Keep exact HTTP status codes and JSON field names/types; allow only additive optional fields.
- Q: How should cache keys/TTL policy be handled in this refactor? → A: Redesign cache keys and TTL policy now; compatibility preservation is not required before production.
- Q: Which model ownership approach should the refactor use? → A: Domain-first canonical DTOs in `internal/futbol`, with provider payloads mapped into them.
- Q: Should route paths and query parameter names change during this feature? → A: Keep current route paths and query parameter names exactly; no API surface renames.
- Q: What should happen when provider calls fail at runtime? → A: Return stale cached data when available; otherwise return error.
- Q: Should this feature include any client/mobile changes? → A: No; backend-only changes.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Stable API behavior after extraction (Priority: P1)

As a mobile client and existing backend consumer, I need the current futbol endpoints to keep the same behavior and payload shape after refactoring so no app feature regresses.

**Why this priority**: Preserving existing production behavior is mandatory for safe refactor delivery.

**Independent Test**: Existing futbol endpoints return equivalent status codes and response structures for representative requests before and after refactor.

**Acceptance Scenarios**:

1. **Given** existing endpoint routes, **When** requests are made with valid parameters, **Then** responses remain schema-compatible with prior behavior.
2. **Given** invalid query/path parameters, **When** requests are made, **Then** error status and message patterns remain consistent.
3. **Given** cache hit and cache miss paths, **When** requests are made, **Then** cache behavior and TTL policy remain unchanged.

---

### User Story 2 - Extracted reusable futbol domain service (Priority: P2)

As a backend engineer, I need futbol API access, transformation, and caching logic in a dedicated package so handlers are thin and logic can be reused outside HTTP handlers.

**Why this priority**: This delivers the core maintainability and reuse objective.

**Independent Test**: New package can be exercised from unit tests without HTTP handler context.

**Acceptance Scenarios**:

1. **Given** a service consumer, **When** it requests fixtures/lineups/stats/standings data, **Then** it can call package APIs without importing handler code.
2. **Given** network or upstream API failures, **When** package methods run, **Then** typed/structured errors are returned and mapped by handlers.
3. **Given** cache is provided, **When** methods are called repeatedly, **Then** package-level cache key and TTL strategy is applied consistently.

---

### User Story 3 - Pluggable provider abstraction (Priority: P3)

As a platform engineer, I need a provider interface in the futbol package so we can swap from API-Football/RapidAPI to another source with minimal handler changes.

**Why this priority**: This is the forward-looking extensibility objective.

**Independent Test**: A fake provider can be injected in tests and handler/service behavior still works.

**Acceptance Scenarios**:

1. **Given** a provider interface, **When** a mock implementation is injected, **Then** package methods and handlers operate without real network calls.
2. **Given** provider-specific request headers/base URL details, **When** wiring is changed in config, **Then** downstream business logic remains unchanged.

---

### Edge Cases

- Mixed legacy migration behavior where helper functions in `futbol.go` are used by non-handler code.
- Partial cache outages where `Exists` succeeds but `Get` fails.
- Upstream API returns malformed JSON or empty-but-success responses.
- Date parsing and season derivation edge cases for league-filtered fixture requests.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST create a dedicated package at `services/api/internal/futbol/` for domain logic currently embedded in `services/api/internal/api/futbol.go`.
- **FR-002**: System MUST keep existing futbol endpoint routes and externally visible response contracts backward-compatible.
- **FR-003**: System MUST expose package APIs for key operations currently tied to handlers (fixtures, lineups, stats, standings, related transformations) without requiring handler imports.
- **FR-004**: System MUST define a provider abstraction so source-specific HTTP details are isolated from business/transform logic.
- **FR-005**: System MUST redesign cache key conventions and TTL policy for maintainability and operational clarity within this feature.
- **FR-006**: System MUST centralize request construction, authentication header handling, and response parsing in the new package.
- **FR-007**: System MUST include unit tests for extracted package logic and integration tests for handler-to-package wiring on high-risk endpoints.
- **FR-008**: System MUST keep path references in docs and generated artifacts repository-relative.
- **FR-009**: System MUST preserve exact existing HTTP status codes and JSON field names/types for current futbol endpoints; only additive optional fields are allowed.
- **FR-010**: System MUST define canonical domain DTOs in `internal/futbol`; provider-specific response payloads MUST be mapped to these DTOs before handler use.
- **FR-011**: System MUST preserve current futbol route paths and existing query parameter names in this feature.
- **FR-012**: System MUST use stale-on-error behavior: on provider failure, return stale cached data when available; if no cache exists, return an error response.
- **FR-013**: System MUST be backend-only; no client/mobile/web code changes are in scope for this feature.

### Key Entities *(include if feature involves data)*

- **FutbolService**: Orchestrates cache lookup, provider fetch, transform, and error mapping for futbol operations.
- **FutbolProvider**: Interface for external data source operations (fixtures, lineup, stats, standings, etc.).
- **FutbolClient**: Concrete API-Football implementation of `FutbolProvider`.
- **CachePolicy**: Key generation and TTL strategy by operation and match status.
- **FutbolResponse DTOs**: Canonical domain DTOs owned by `internal/futbol` and used by service/handler boundaries.
- **Provider Payload DTOs**: API-source-specific structs scoped to provider implementation and not leaked as handler contracts.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `services/api/internal/api/futbol.go` is reduced to handler and routing glue, with core logic moved to `services/api/internal/futbol/`.
- **SC-002**: Existing futbol endpoints pass regression checks for representative success/error/cache scenarios with no contract-breaking changes.
- **SC-003**: At least one package-level unit test suite validates provider + cache orchestration independent of HTTP handlers.
- **SC-004**: Switching provider implementation requires only wiring/config changes and no endpoint contract rewrite.
- **SC-005**: Baseline endpoint contract tests confirm status-code and response-schema parity pre/post-refactor (except additive optional fields).
- **SC-006**: Runtime failure tests confirm stale cache fallback is returned when provider calls fail, and error is returned only when cache is unavailable.
- **SC-007**: Diff scope for implementation PR contains backend files only (primarily under `services/api/**` and `specs/019-modularize-futbol-package/**`).
