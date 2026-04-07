# Quickstart - Modularize Futbol Backend Package

## Goal

Refactor `services/api/internal/api/futbol.go` into `services/api/internal/futbol/` while preserving API behavior.

## Scope Guardrail

- Backend-only feature: modify backend code/docs only.
- Do not change client/mobile/web code (for example under `apps/mobile/`).

## Prerequisites

- Go toolchain from `services/api/go.mod`
- Existing API env vars configured (`FOOTBALL_API_KEY`, optional `API_FOOTBALL_BASE_URL`, cache settings)
- Existing tests runnable via `go test`

## Refactor Sequence

1. Create package skeleton:
   - `services/api/internal/futbol/client.go`
   - `services/api/internal/futbol/service.go`
   - `services/api/internal/futbol/cache.go`
   - `services/api/internal/futbol/transformer.go`
   - `services/api/internal/futbol/types.go`
2. Move reusable methods first (currently used outside handlers):
   - lineup/stats/standings/head-to-head fetchers
   - standings summary formatter
3. Keep handler signatures/routes unchanged in `services/api/internal/api/futbol.go`.
4. Switch handlers to call `futbol.Service` methods.
5. Redesign cache key namespace and TTL policy in `internal/futbol/cache.go` (with tests), while keeping endpoint response contracts stable.
6. Add provider interface and API-Football implementation.
7. Run regression tests and endpoint checks.

## Validation Commands

From repo root:

```bash
cd services/api
go test ./internal/api -run Futbol -count=1
go test ./internal/futbol/... -count=1
go test ./... -count=1
```

## Manual HTTP Smoke Tests

Assuming local API is running:

```bash
curl "http://localhost:8080/v1/api/futbol/matches?date=2026-04-07"
curl "http://localhost:8080/v1/api/futbol/lineup?match_id=12345"
curl "http://localhost:8080/v1/api/futbol/leagues"
curl "http://localhost:8080/v1/api/futbol/team_standings?team_id=33"
curl "http://localhost:8080/v1/api/futbol/league_standings?league_id=39&season=2025"
```

## Expected Outcomes

- Endpoint paths and JSON shape remain compatible with current clients.
- `internal/api/futbol.go` is reduced to request validation + response mapping.
- Reusable futbol logic is importable from `internal/futbol`.
- Debate aggregator usage remains functional through service-level APIs.
- No client/mobile/web files are changed as part of this feature.
<<<<<<< HEAD

## Validation Results

Validated on 2026-04-07:

- `go test ./internal/api -run Futbol -count=1` -> pass
- `go test ./internal/futbol/... -count=1` -> pass
- `go test ./... -count=1` -> pass

## Scope Verification

- Command: `git diff --name-only main...HEAD -- apps/mobile apps/admin`
- Result: no changed files under `apps/mobile` or `apps/admin`
=======
>>>>>>> 0769677f (fixed conflict)
