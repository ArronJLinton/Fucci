# Data Model: AI Powered Debate Generator

**Feature**: 004-ai-debate-generator  
**Date**: 2025-02-15  
**Phase 1 output**  
**Aligned with**: [spec.md](./spec.md), [user-stories.md](./user-stories.md)

## Purpose

This document describes the entities and data flow for the AI Powered Debate Generator: debate types (Pre Match, Post Match), the **Context Bundle** (head-to-head history, league table, news, form, stats, etc.), and how they feed the AI prompt and stored debate. It supports Epics A–H in user-stories.md.

## Entity Overview

| Entity            | Purpose |
|-------------------|--------|
| Debate            | Stored AI-generated debate (match_id, phase/type, headline, description, cards; optional safety_classification, context_quality, source_context_ids) |
| DebateCard        | Single stance (agree/disagree/wildcard or side_a/side_b) with title, description; optional topic_tags |
| Debate Thread     | Debate + metadata + comments + votes (logical view for delivery) |
| Context Bundle    | Structured inputs to the generator: match metadata, H2H, form, players/stats, injuries, news (see below) |
| MatchData         | In-memory aggregated input to the AI (match info + H2H + league table + news + optional lineups/stats/sentiment) |
| HeadToHeadHistory | Past fixtures between the two teams (from API-Football) |
| LeagueTable       | Standings for the match’s league/season (from API-Football or internal) |
| NewsHeadlines     | Relevant news items (from existing news client); optional source IDs for traceability |

## Data Flow

```
Match (fixture id, teams, league, season, status)
    │
    ▼
DebateDataAggregator.AggregateMatchData()
    │
    ├── fetchHeadToHead(homeTeamID, awayTeamID)     → HeadToHeadHistory  [NEW]
    ├── fetchLeagueStandings(leagueID, season)       → LeagueTable        [NEW]
    ├── fetchNewsHeadlines(homeTeam, awayTeam)      → NewsHeadlines       [existing]
    ├── fetchLineups(matchID)                        → Lineups            [existing, optional]
    ├── fetchMatchStats(matchID)                     → Stats             [existing, optional]
    └── fetchSocialSentiment(...)                    → SocialSentiment    [existing, optional]
    │
    ▼
ai.MatchData (extended with H2H + LeagueTable)
    │
    ▼
PromptGenerator.buildUserPrompt(matchData, "pre_match" | "post_match")
    │
    ▼
OpenAI → DebatePrompt { Headline, Description, Cards[] }
    │
    ▼
Persist: debates + debate_cards (+ analytics)
```

### Multi-debate flow (generate set)

When generating **multiple debates per type** (e.g. 3 pre_match, 3 post_match):

1. **One AI call per type**: Prompt asks for an **array** of N debate prompts (e.g. 3). Response shape: `[{ headline, description, cards }, ...]`.
2. **Cache**: One key per (match, type) storing the **full set** of debates, e.g. `pre_match_debates:{matchID}` or `debates:{matchID}:pre_match`. Value: array of debate payloads (or list of debate IDs after persist). TTL as per existing policy (e.g. 24h).
3. **DB**: **One row per debate**; multiple rows per `(match_id, debate_type)` are expected. No uniqueness constraint on (match_id, debate_type); list endpoints return all debates for the match (optionally filtered by type).
4. **Preload**: When the user opens Match Details, the client or API can trigger “generate set” in the background for the applicable type so that by the time the user opens the Debate tab, the set is often already available.

## Source Details

### 1. Head-to-head history

- **Source**: API-Football `GET /fixtures/headtohead?h2h={home_team_id}-{away_team_id}&last=10`
- **Cache**: Redis key `h2h:{home_id}-{away_id}`, TTL e.g. 6–12 hours
- **Shape**: List of past fixtures (date, home/away teams, score, winner). Passed into `MatchData` as structured text or a small struct for the prompt builder.
- **Resolution**: Team IDs from fixture response (`teams.home.id`, `teams.away.id`). If fixture lacks team IDs, H2H is skipped (graceful degradation).

### 2. League table

- **Source**: Internal `getLeagueStandingsByLeagueId(league_id, season)` or direct API-Football `GET /standings?league=&season=`
- **Cache**: Existing `league_standings:{league_id}:{season}`
- **Shape**: Standings rows (rank, team name, points, etc.). For the prompt, include at least both teams’ rank and points (and optionally a short summary).
- **Resolution**: `league_id` and `season` from fixture. If missing, league table is skipped (graceful degradation).

### 3. News articles

- **Source**: Existing news client / aggregator `fetchNewsHeadlines` (team names + "Home vs Away" query).
- **Shape**: `[]string` headlines in `MatchData.NewsHeadlines` (unchanged).
- **Count**: Current behaviour (e.g. per-team + matchup); no change required for this spec.

## MatchData (AI package)

Current `MatchData` in `services/api/internal/ai/prompt_generator.go`:

- MatchID, HomeTeam, AwayTeam, Date, Status, Venue, League, Season
- Lineups, Stats, NewsHeadlines, SocialSentiment

**Add**:

- `HeadToHeadSummary string` or `HeadToHead *HeadToHeadHistory` – text or struct describing last N meetings (date, score, outcome).
- `LeagueTableSummary string` or `LeagueTable *LeagueTableInfo` – text or struct with both teams’ rank and points (and optionally league name).

Prompt builder must include dedicated sections:

- `HEAD-TO-HEAD:` (or `H2H:`)
- `LEAGUE TABLE:` (or `STANDINGS:`)
- `NEWS HEADLINES:` (existing)

## Context Bundle (Epic B1)

The **Context Bundle** is the structured set of inputs passed to the generator. It includes (even if some fields are empty):

- **match metadata**: teams, venue, competition, kickoff time, status
- **H2H summary**: last X meetings (date, score, outcome)
- **team form**: last X matches per team (optional)
- **key players + stats**: goals, assists, ratings if available (optional)
- **injuries/suspensions**: if available (optional)
- **recent news**: headlines + snippets + source IDs

Bundle is cached with TTL (e.g. by match_id + phase); see Epic F1. Debates created with partial context are tagged **context_quality** = partial (Epic B2).

## Debate and DebateCard (persisted)

- **Debate**: match_id, debate_type / phase (`pre_match` | `post_match`), headline, description, ai_generated. Optional extensions for phased work: safety_classification (approved | needs_review | blocked), context_quality (full | partial), source_context_ids (refs to stat/news used). Stored in `debates` table (existing; schema extensions as needed for E1, B2). **Multiple rows per (match_id, debate_type)** are allowed when generating a set (e.g. 3 debates per type).
- **DebateCard**: debate_id, stance, title, description, ai_generated. Optional: side_a_label, side_b_label, topic_tags. Stored in `debate_cards` table (existing; schema extensions as needed for A1).
- **Debate Thread** (logical): Debate + comments + votes; used for delivery (Epic C) and ranking (Epic D3).
- **Cache (multi-debate)**: One key per (match_id, debate_type) storing the full set, e.g. `pre_match_debates:{matchID}`. Value: array of debate payloads or list of debate IDs; TTL as per existing policy (e.g. 24h).

## Validation Rules

- **Pre Match**: Only when fixture status is NS (or equivalent “not started”). Reject post_match for NS.
- **Post Match**: Only when fixture status is FT, AET, or PEN. Reject pre_match for finished matches.
- **Debate prompt**: Must have non-empty headline and at least one card. If OpenAI returns invalid structure, return 500 and do not persist.

## State Transitions

- **Generate (pre_match)**: Match NS → aggregate (H2H, table, news, optional lineups) → prompt → persist debate + cards.
- **Generate (post_match)**: Match FT/AET/PEN → aggregate (H2H, table, news, stats, optional sentiment) → prompt → persist debate + cards.
- **Re-generate**: If debate already exists for (match_id, type), either return existing or soft-delete and regenerate when `force_regenerate=true`.
