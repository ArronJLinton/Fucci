# Research: AI Powered Debate Generator

**Feature**: 004-ai-debate-generator  
**Phase 0 output** – Resolves NEEDS CLARIFICATION and documents integration choices.

## 1. Head-to-head history source

**Decision**: Use API-Football (RapidAPI) Head to Head fixtures endpoint.

**Endpoint**: `GET .../fixtures/headtohead?h2h={home_team_id}-{away_team_id}` (and optional `last=N` to limit results).

**Rationale**:

- Spec requires "history between both types" (head-to-head) as a source.
- Existing Fucci stack already uses API-Football for fixtures, lineups, standings; same key and base URL.
- H2H parameter format: team IDs separated by hyphen (e.g. `33-34`). Team IDs come from the fixture response (teams.home.id, teams.away.id).
- Optional `last=10` (or similar) keeps payload and prompt size bounded.

**Alternatives considered**:

- Storing H2H in our DB: Rejected for Phase 1; external API is single source of truth and avoids sync logic.
- Another provider for H2H: Rejected to avoid extra keys and inconsistency with existing football data.

**Implementation note**: Add a fetcher in `DebateDataAggregator` (or shared football client) that calls the H2H endpoint; cache by `h2h:{home_id}-{away_id}` with TTL (e.g. 6–12h) to reduce API usage.

---

## 2. League table source

**Decision**: Use existing internal league standings API and call it from the debate aggregator.

**Current state**: `getLeagueStandingsByLeagueId` in `services/api/internal/api/futbol.go` returns standings for a given `league_id` and `season`. Cache key: `league_standings:{league_id}:{season}`.

**Rationale**:

- Spec requires "league table" as a source; standings are already available and cached.
- For a given fixture we have (or can obtain) `league.id` and `league.season` from the fixture; use those to request standings.
- Aggregator can call the same internal football API or the same RapidAPI standings endpoint with the same cache semantics.

**Alternatives considered**:

- Calling RapidAPI standings directly from the aggregator: Acceptable and consistent with other aggregator fetchers; prefer reusing existing cache key pattern if calling internal handler is not feasible from aggregator (e.g. same process, so direct function or internal HTTP call).

**Implementation note**: From fixture, resolve `league_id` and `season`; fetch standings (internal or direct API); add a `LeagueTable` or `Standings` structure to `MatchData` and include in the prompt.

---

## 3. News articles source

**Decision**: Keep using the existing news path in `DebateDataAggregator` (e.g. `fetchNewsHeadlines` / search by team and "Home vs Away" query).

**Rationale**:

- Already implemented; spec lists "News Articles" as a required source.
- No change to source choice; only ensure it is always requested and passed into the prompt (and document as one of the three required sources).

**Alternatives considered**: None; current implementation is sufficient.

---

## 4. Prompt design (H2H + league table + news)

**Decision**: Extend the AI `MatchData` and `buildUserPrompt` to include:

- **Head-to-head**: Summary of last N meetings (e.g. date, score, winner or "draw"). Format as a short list or paragraph so the model can reference "last 5 meetings", "form in H2H", etc.
- **League table**: Top/bottom positions and points for the two teams (and optionally a one-line summary, e.g. "3rd vs 7th"). Avoid dumping full table unless needed.
- **News**: Already present as `NewsHeadlines`; keep and ensure they are clearly labeled (e.g. "NEWS HEADLINES") in the user prompt.

**Rationale**:

- Pre-match: H2H + table + news give context for "who’s in form", "stakes", "narrative".
- Post-match: Same three plus result/stats allow "result in context of H2H/table" and "narrative from news".
- Structured sections in the prompt (H2H, LEAGUE TABLE, NEWS) reduce ambiguity and improve consistency.

**Alternatives considered**:

- Single unstructured blob: Rejected; structured sections improve reliability and debugging.

---

## 5. Graceful degradation

**Decision**: If H2H or league table fetch fails (or returns empty), still call the AI with the remaining sources; do not fail the whole request unless all three fail or the match context is missing.

**Rationale**:

- Spec FR-007: "SHOULD degrade gracefully when one or more sources are temporarily unavailable".
- Better to return a debate with "News only" or "News + table" than to return 500 when H2H is down.

**Implementation note**: Aggregator returns partial `MatchData`; prompt builder omits missing sections; log missing sources for observability.

---

## 6. Clarifications resolved

| Spec question                         | Resolution |
|--------------------------------------|------------|
| How many H2H meetings to include?    | Use `last=10` (configurable); document in data-model. |
| Which league table to use?           | Match’s competition: `league_id` and `season` from fixture. |
| How many news items?                 | Keep current behaviour (e.g. per-team + matchup query); document in data-model. |
