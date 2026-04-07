# Data Model - Modularize Futbol Backend Package

## Overview

This feature is a backend-only refactor-oriented model for code structure and runtime DTO flow, not a database schema change.

## Entities

### 1) FutbolService

- **Purpose**: Orchestrates cache lookup, provider fetch, transform, and error mapping for futbol operations.
- **Core fields/dependencies**:
  - `provider` (`FutbolProvider`)
  - `cache` (`cache.CacheInterface`)
  - `clock` (optional for TTL/testing determinism)
- **Key operations**:
  - `GetMatches(date, leagueID)`
  - `GetLineup(matchID)`
  - `GetMatchStats(matchID)`
  - `GetLeagues(season)`
  - `GetTeamStandings(teamID, season)`
  - `GetLeagueStandings(leagueID, season)`
  - `GetHeadToHead(homeTeamID, awayTeamID)`

### 2) FutbolProvider (interface)

- **Purpose**: Abstracts external data source operations from domain/service logic.
- **Contract responsibilities**:
  - Build/fetch external requests
  - Return provider-layer DTOs or domain DTOs
  - Surface upstream failures as typed errors
- **Initial implementation**: API-Football provider with RapidAPI headers.

### 3) FutbolClient (API-Football implementation)

- **Purpose**: Concrete adapter for current external API.
- **Core fields**:
  - `baseURL`
  - `apiKey`
  - HTTP transport utility dependency
- **Validation rules**:
  - Non-empty API key
  - 2xx status required for success path
  - JSON parsing failures surfaced as parse errors

### 4) CachePolicy

- **Purpose**: Encapsulates cache key conventions and TTL selection logic currently spread across handlers.
- **Key fields**:
  - key prefixes by operation
  - status-to-TTL mapping for match-centric endpoints
- **Rules**:
  - Redesign key namespace for maintainability (documented and test-covered)
  - TTL policy is operation/status-aware with stale-on-error support

### 5) FutbolResponse DTOs

- **Purpose**: Stabilize response shaping between provider and handler layers.
- **Examples**:
  - Matches response DTO
  - Lineup DTO
  - Stats DTO
  - Standings DTO
  - League DTO
- **Rules**:
  - HTTP response compatibility maintained for existing endpoints
  - Provider-specific raw structs not leaked into handler package where avoidable

## Relationships

- `api handlers` -> `FutbolService`
- `FutbolService` -> `FutbolProvider` + `CachePolicy` + `cache.CacheInterface`
- `FutbolProvider` -> external API transport
- `debate_data_aggregator` -> `FutbolService` reusable methods (not handler internals)
- client/mobile layers remain unchanged in this feature

## State Transitions

### Request lifecycle

1. Handler validates request parameters.
2. Service computes cache key and checks cache.
3. On hit: return cached DTO.
4. On miss: provider fetch + parse.
5. Service transforms/normalizes and stores cache with policy TTL.
6. Handler maps result/error to HTTP response.

### Error lifecycle

1. Validation errors originate in handler or service input guards.
2. Provider/network errors are wrapped as typed upstream errors.
3. Service may attempt stale/fallback strategy where applicable.
4. Handler maps final error class to status code and response message.
