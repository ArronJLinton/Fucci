# Phase 0 Research - Modularize Futbol Backend Package

## Decision 1: Mirror `internal/news` package boundary for futbol

- **Decision**: Implement `services/api/internal/futbol/` with separate files for client/service/cache/transform helpers, while keeping `services/api/internal/api/futbol.go` as thin HTTP handlers.
- **Rationale**: `internal/news` already demonstrates stable handler-to-domain layering in this codebase and reduces coupling to transport details.
- **Alternatives considered**:
  - Keep `futbol.go` monolithic and add comments only: rejected because coupling and testability issues remain.
  - Split into many micro-packages immediately: rejected as higher migration risk than one cohesive domain package.

## Decision 2: Introduce provider abstraction at domain boundary

- **Decision**: Define a provider interface in `internal/futbol` for external data operations (fixtures/lineups/stats/standings/head-to-head), with API-Football as initial implementation.
- **Rationale**: Directly meets the goal of future source swapping with minimal handler changes.
- **Alternatives considered**:
  - Hardcode API-Football client calls in service methods: rejected due to vendor lock-in.
  - Abstract at handler layer only: rejected because business logic would still be coupled to transport DTOs.

## Decision 3: Redesign cache keys and TTL policy during extraction

- **Decision**: Redesign cache keys and TTL policy in `internal/futbol` now (pre-production), while keeping endpoint contracts stable.
- **Rationale**: Improves maintainability and operational clarity without migration risk from production consumers.
- **Alternatives considered**:
  - Preserve legacy key namespace and TTLs: rejected because team explicitly chose cleanup now.
  - Remove cache during refactor: rejected due to avoidable performance risk.

## Decision 4: Keep debate aggregator compatibility through service-level APIs

- **Decision**: Move currently reused methods (`FetchLineupData`, `FetchMatchStatsData`, `GetLeagueStandingsData`, `FetchHeadToHead`, summary helpers) into service APIs consumed by both handlers and aggregator logic.
- **Rationale**: Existing cross-feature usage in debate data aggregation is a hard dependency and must remain stable.
- **Alternatives considered**:
  - Duplicate logic in aggregator and handlers: rejected due to drift and maintenance burden.
  - Force aggregator to consume HTTP endpoints: rejected due to inefficiency and coupling.

## Decision 5: Testing strategy combines endpoint regression and new unit tests

- **Decision**: Keep current handler tests as regression guardrails and add focused unit tests in `internal/futbol` for orchestration/cache/provider behavior.
- **Rationale**: Refactors need both black-box safety and white-box confidence in extracted logic.
- **Alternatives considered**:
  - Handler tests only: rejected because new package internals would be under-tested.
  - Unit tests only: rejected because HTTP contract compatibility could regress unnoticed.

## Resolved Clarifications

- **Runtime stack**: Go backend service (`services/api`) with existing cache abstraction and custom HTTP utility.
- **Integration pattern**: Handler layer remains in `internal/api`; domain package handles provider/caching/transform behavior.
- **Non-functional constraints**: Preserve endpoint contracts and route/query naming; backend-only scope (no client/mobile/web code changes).
