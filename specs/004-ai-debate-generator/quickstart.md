# Quickstart: AI Powered Debate Generator

**Feature**: 004-ai-debate-generator  
**Date**: 2025-02-15  
**Target**: Developers running and testing debate generation locally  
**See also**: [spec.md](./spec.md), [user-stories.md](./user-stories.md) for full scope and Epics A–H.

## Prerequisites

- Go 1.22+, Node 18+ (if testing mobile)
- PostgreSQL and Redis (as per main Fucci API)
- **OpenAI API key** (required for AI generation)
- **RapidAPI key** for API-Football (fixtures, standings, and—once implemented—head-to-head)
- Backend and (optional) news service configured per main repo README

## Environment

Ensure the API has:

- `OPENAI_API_KEY` – used by the prompt generator
- Football API key (e.g. RapidAPI) – used for fixture, standings, and H2H (when added)
- News API – existing news client used for the “news articles” source

## Run the API

From repo root:

```bash
# Start dependencies (PostgreSQL, Redis) if not already running
# Then from services/api or as per your setup:
go run ./cmd/api
# Or: make run (if defined)
```

Default base URL: `http://localhost:8080`. Debate routes are under `/v1/api/debates`.

## Test debate generation

### 1. Generate prompt only (no persist)

Returns the AI prompt (headline + cards) without saving. Use this to verify that the three sources (H2H, league table, news) are reflected once implemented.

**Pre-match** (use a fixture ID that is scheduled, not started):

```bash
curl -s "http://localhost:8080/v1/api/debates/generate?match_id=1234567&type=pre_match" | jq .
```

**Post-match** (use a fixture ID that has finished):

```bash
curl -s "http://localhost:8080/v1/api/debates/generate?match_id=1234567&type=post_match" | jq .
```

Expected: `200` and JSON with `headline`, `description`, and `cards[]` (each with `stance`, `title`, `description`). If `match_id` is invalid or status doesn’t match type, you may get `200` with an `info` message or `400`/`500`.

### 2. Generate and persist debate

Creates a debate in the DB and returns the full debate with cards:

```bash
curl -s -X POST "http://localhost:8080/v1/api/debates/generate" \
  -H "Content-Type: application/json" \
  -d '{"match_id":"1234567","debate_type":"post_match"}' | jq .
```

Optional: `"force_regenerate": true` to replace an existing debate for that match and type.

### 3. Get debate by ID

```bash
curl -s "http://localhost:8080/v1/api/debates/1" | jq .
```

### 4. Get debates by match

```bash
curl -s "http://localhost:8080/v1/api/debates/match?match_id=1234567" | jq .
```

## Data sources (after implementation)

Once the aggregator is extended per this spec:

| Source           | Used for              | When fetched                         |
|------------------|------------------------|--------------------------------------|
| Head-to-head      | H2H history            | Both pre_match and post_match        |
| League table      | Standings (rank, pts)  | Both; league/season from fixture     |
| News articles     | Headlines/snippets     | Both; existing news client           |

Optional (already or to be aggregated): lineups, match stats, social sentiment.

## Troubleshooting

- **501 Not Implemented** – `AIPromptGenerator` is nil; set `OPENAI_API_KEY` and ensure the API config wires the prompt generator.
- **500 from generate** – Check logs for aggregation errors (e.g. fixture not found, football API key, or OpenAI errors).
- **Empty or missing sections in prompt** – Verify H2H and league table fetchers are implemented and that fixture has `league_id`/`season` and team IDs for H2H.

## Contract

OpenAPI spec: [contracts/api.yaml](./contracts/api.yaml). Paths:

- `GET /api/debates/generate?match_id=&type=`
- `POST /api/debates/generate`
- `GET /api/debates/{id}`
- `GET /api/debates/match?match_id=`

(Base URL in spec is `/v1`; so full path is e.g. `http://localhost:8080/v1/api/debates/...`.)
