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
- `JWT_SECRET` – required for auth (generate with: `openssl rand -base64 32`)
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

### 3. Generate set (multiple debates per type)

Creates multiple debates (e.g. 3) for the match and type in one AI call, persists each, and returns the full set. Use for preload (when user opens match details) or when the Debate tab finds no debates:

```bash
curl -s -X POST "http://localhost:8080/v1/api/debates/generate-set" \
  -H "Content-Type: application/json" \
  -d '{"match_id":"1234567","debate_type":"pre_match","count":3}' | jq .
```

Response: `{ "debates": [ DebateResponse, ... ] }` (and optional `"pending": true` if generation is async). Optional `"force_regenerate": true` replaces existing debates for that match and type.

### 4. Get debate by ID

```bash
curl -s "http://localhost:8080/v1/api/debates/1" | jq .
```

### 5. Get debates by match

```bash
curl -s "http://localhost:8080/v1/api/debates/match?match_id=1234567" | jq .
```

Optional query: `debate_type=pre_match` or `debate_type=post_match` to filter. When multiple debates per type are supported, the list may contain several items per type.

## Data sources (after implementation)

Once the aggregator is extended per this spec:

| Source           | Used for              | When fetched                         |
|------------------|------------------------|--------------------------------------|
| Head-to-head      | H2H history            | Both pre_match and post_match        |
| League table      | Standings (rank, pts)  | Both; league/season from fixture     |
| News articles     | Headlines/snippets     | Both; existing news client           |

Optional (already or to be aggregated): lineups, match stats, social sentiment.

## Troubleshooting

- **JWT_SECRET is not set** – Add `JWT_SECRET` to your `.env`. Generate a value: `openssl rand -base64 32`. Without it, login and protected routes will not work.
- **News API 403 "You are not subscribed to this API"** – Your RapidAPI key does not have an active subscription to the Google News API. Subscribe in the RapidAPI dashboard, or ignore; debate generation still runs with H2H and league table only.
- **News API 429 "Too many requests"** – Rate limit hit (e.g. free tier). Wait before retrying or upgrade the news API plan. Debates are still generated with other context when news fails.
- **501 Not Implemented** – `AIPromptGenerator` is nil; set `OPENAI_API_KEY` and ensure the API config wires the prompt generator.
- **500 from generate** – Check logs for aggregation errors (e.g. fixture not found, football API key, or OpenAI errors).
- **Empty or missing sections in prompt** – Verify H2H and league table fetchers are implemented and that fixture has `league_id`/`season` and team IDs for H2H.

## Preload (multi-debate)

When the user opens **Match Details**, the client can call `POST /api/debates/generate-set` in the background for the applicable `debate_type` (pre_match if match not finished, post_match if finished). By the time they open the Debate tab, the set is often already in cache/DB and can be shown immediately; if still generating, show a single loading state until the full set is ready.

## Contract

OpenAPI spec: [contracts/api.yaml](./contracts/api.yaml). Paths:

- `GET /api/debates/generate?match_id=&type=`
- `POST /api/debates/generate`
- `POST /api/debates/generate-set` — generate multiple debates (e.g. 3) per match and type; returns `{ debates: DebateResponse[] }`
- `GET /api/debates/{id}`
- `GET /api/debates/match?match_id=` (optional `debate_type=`)

(Base URL in spec is `/v1`; so full path is e.g. `http://localhost:8080/v1/api/debates/...`.)
