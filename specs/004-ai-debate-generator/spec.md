# Feature Specification: AI Powered Debate Generator

**Feature Branch**: `004-ai-debate-generator`  
**Created**: 2025-02-15  
**Status**: Draft  
**Input**: Dedicated spec for the AI Powered Debate Generator. Debate types: Pre Match, Post Match. Sources: Head-to-head history, League table, News articles. Aligned with full User Stories (Epics A–H).

## Relationship to Other Specs

This feature is referenced in **001-football-community** (User Story 2: AI-Powered Debate Engagement). This spec defines the debate generator in detail: debate types, data sources, roles, epics, and behaviour. Full user stories and acceptance criteria are in [user-stories.md](./user-stories.md).

## Scope

Generate, deliver, and manage AI-created match debates using structured football context (H2H, form, player stats, news) and enable fan participation (comments + voting) with safety, caching, and moderation.

## Roles

| Role | Description |
|------|-------------|
| **Regular User (Fan)** | Views debates, comments, votes; triggers on-demand generation when debates are missing |
| **Team Manager** | (Per 001-football-community; no debate-specific permissions in this spec) |
| **Admin/Moderator** | Reviews/blocks AI prompts; moderates comments; configures sources and generator params; views audit and metrics |
| **System (Jobs/Workers)** | Builds context bundles; runs generation (sync or async); applies caching, rate limits, deduplication |

## Definitions

- **Debate Prompt**: A two-sided question framing a polarizing but safe discussion (headline + optional description + cards with stances).
- **Debate Thread**: Prompt + metadata + comments + votes; the full unit shown to users.
- **Context Bundle**: Structured match/team/player/news inputs provided to the generator (see [data-model.md](./data-model.md)).

## Goals

- Generate AI-powered debate prompts for football matches to drive fan engagement.
- Support **Pre Match** debates (before kick-off) and **Post Match** debates (after the match).
- Use a defined set of **sources** so debates are grounded in real data: head-to-head history, league table, and news articles.
- Support fan participation (comments, voting), safety/moderation, caching, and admin controls as defined in Epics A–H.

## Debate Types

| Type        | When available        | Purpose |
|------------|------------------------|--------|
| **Pre Match**  | Before kick-off (match status NS/scheduled) | Build anticipation using form, H2H, table position, and recent news |
| **Post Match** | After full-time (FT, AET, PEN)             | Drive discussion using result, stats, key moments, and post-match narrative |

## Data Sources (Required Inputs)

The generator MUST use the following sources when producing debate prompts:

1. **Head-to-head history** – Past results and meetings between the two teams (e.g. last N meetings, goals, outcomes). Used for both pre- and post-match context.
2. **League table** – Current standings (points, position, form) for the competition. Used for pre-match context (rivalry, stakes) and post-match (impact on table).
3. **News articles** – Relevant headlines/items about the teams, the fixture, or key players. Used for narrative and controversy in both debate types.

Additional sources (e.g. lineups, match stats, team form, social sentiment) may be used where available (Context Bundle — see [data-model.md](./data-model.md)).

## Epics Overview

| Epic | Name | MVP | Description |
|------|------|-----|--------------|
| **A** | Debate Generation (Core AI) | A1, A2, A3, A5 | Pre-match generation, on-demand fallback, post-match narratives, quality guardrails; A4 (variety/deduplication) post-MVP |
| **B** | Context Building | B1, B2 | Context bundle (H2H, form, stats, news), graceful degradation; B3 (admin source control) post-MVP |
| **C** | Delivery in Product | C1 | Debates on match details screen; C2 (sorting/trending), C3 (deep links) post-MVP |
| **D** | Engagement (Comments + Voting) | D1, D2, D3 | Comment, upvote/downvote, thread ranking |
| **E** | Safety & Moderation | E1, E2 | Moderate AI prompts, moderate comments; E3 (abuse prevention) post-MVP |
| **F** | Cost Control & Performance | F1, F3 | Cache by match + phase; async job processing; F2 (rate-limit AI) post-MVP |
| **G** | Admin & Observability | G1, G2, G3 | Config panel/table, audit trail, metrics dashboard — post-MVP or phased |
| **H** | Future Enhancements | — | Personalization, multi-language, AI referee summarization |

**MVP cut (leanest)**  
Prioritise: **A1, B1, C1** (generate + context + display), **D1, D2** (comment + vote), **E1, E2** (moderation basics), **F1, F3** (caching + async worker). See [user-stories.md](./user-stories.md) for full acceptance criteria.

## User Scenarios & Testing _(mandatory)_

### User Story 1 – Pre Match Debate (Priority: P1) — maps to Epic A1

Users can view a set of AI-generated debates for a match before kickoff so they can join discussions early.

**Acceptance Scenarios**:

1. **Given** a match is scheduled (not started), **When** the user requests or views pre-match debates, **Then** the system returns debate prompt(s) (headline + cards) that reference H2H, table, and news where available.
2. **Given** H2H or table data is unavailable, **When** the user requests a pre-match debate, **Then** the system still returns a debate using other available sources (e.g. news only) with graceful degradation.
3. **Given** a pre-match debate is generated, **When** the user views it, **Then** they see a clear headline and multiple debate cards (e.g. agree/disagree/wildcard or side_a / side_b).

### User Story 2 – On-Demand Generation (Priority: P1) — maps to Epic A2

If debates are missing for a match, the system generates them on demand so the experience never feels empty.

**Acceptance Scenarios**:

1. **Given** debates for `match_id` are missing, **When** the user or client requests debates, **Then** the API triggers generation and returns either a “pending” state with polling or a minimal default/template set immediately.
2. On-demand generation is rate-limited per match and completes within target SLA (e.g. &lt; 10–20s) or returns a graceful fallback.

### User Story 3 – Post Match Debate (Priority: P1) — maps to Epic A3

Users can view debates that update after the match based on outcome and key moments.

**Acceptance Scenarios**:

1. **Given** a match has finished (FT/AET/PEN), **When** the user requests a post-match debate, **Then** the system returns a debate prompt that references the result, H2H, table impact, and news.
2. **Given** a post-match debate is generated, **When** the user views it, **Then** they see a headline and cards that reflect the match outcome and key talking points.
3. Post-match debates are stored separately or versioned by phase (pre | post).

### User Story 4 – Context & Degradation (Priority: P1) — maps to Epic B1, B2

The system compiles a context bundle (H2H, form, stats, news) and degrades gracefully when data is missing.

**Acceptance Scenarios**:

1. Context bundle includes match metadata, H2H summary, team form, key players/stats, injuries (if available), and recent news; bundle is cached with TTL.
2. Generator can produce debates with partial context; missing fields do not break generation; debates under partial data are tagged (e.g. `context_quality=partial`).

### User Story 5 – Display & Engagement (Priority: P1) — maps to Epic C1, D1, D2

Debates are visible on the match page; fans can comment and vote.

**Acceptance Scenarios**:

1. Debates appear in a dedicated “Debates” tab on the match details screen; empty state explains if debates are still generating.
2. Auth required to comment; comments support basic text; rate limiting prevents spam.
3. Users can upvote/downvote comments; vote state is persisted and reflected in ranking.

### User Story 6 – Safety & Moderation (Priority: P1) — maps to Epic E1, E2

AI-generated prompts and user comments are moderatable; blocked content is not shown.

**Acceptance Scenarios**:

1. Each debate has a safety classification (e.g. approved / needs_review / blocked); admin can approve/block/edit; blocked debates are not shown.
2. Users can report comments; admin review queue shows context; admin actions: delete, warn, suspend/ban.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST support two debate types: **Pre Match** and **Post Match**.
- **FR-002**: System MUST use **head-to-head history** between the two teams as an input when generating debate prompts.
- **FR-003**: System MUST use **league table** (standings for the match’s competition) as an input when generating debate prompts.
- **FR-004**: System MUST use **news articles** (relevant to teams or fixture) as an input when generating debate prompts.
- **FR-005**: System MUST generate a structured debate prompt (e.g. headline, description, debate cards with optional side labels and topic tags) suitable for display in the app.
- **FR-006**: System MUST restrict pre-match debates to matches that have not started and post-match debates to matches that have finished (FT/AET/PEN).
- **FR-007**: System SHOULD degrade gracefully when one or more sources (H2H, table, news) are temporarily unavailable (e.g. return debate with available data or clear messaging).
- **FR-008**: System SHOULD support on-demand generation when debates are missing (pending or immediate fallback), with rate limiting per match.
- **FR-009**: System SHOULD cache generated debates by match + phase (and optionally context_version) with configurable TTL.
- **FR-010**: System SHOULD support comments and voting on debate threads; auth required for posting; moderation and reporting as per Epic E.

### Key Entities

- **Debate**: AI-generated discussion prompt tied to a match and type (pre_match | post_match), with headline, description, cards, and optional safety_classification, context_quality, source_context_ids.
- **DebateCard**: Single stance/topic (e.g. agree, disagree, wildcard or side_a / side_b) with title and description; optional topic_tags.
- **Debate Thread**: Debate + metadata + comments + votes.
- **Context Bundle**: Match metadata, H2H summary, team form, key players/stats, injuries, news (see data-model).
- **Match**: Fixture with teams, date, status, competition (used to resolve H2H and league table).
- **HeadToHeadHistory**, **LeagueTable**, **NewsArticle**: As in data-model.

## Success Criteria _(mandatory)_

- **SC-001**: Pre-match and post-match debates are generated using all three sources (H2H, league table, news) when APIs are available.
- **SC-002**: Debate prompts are available within an agreed SLA (e.g. &lt; 30s sync or &lt; 20s async) after data aggregation.
- **SC-003**: When a source fails, the system either returns a partial debate or a clear error without crashing; partial debates tagged appropriately.
- **SC-004**: Generated content adheres to existing AI guidelines (001-football-community: bias detection, community feedback, fair/unbiased prompts).
- **SC-005**: Debates are cacheable by match + phase; on-demand generation is rate-limited; moderation (E1, E2) allows blocking and reporting.

## Clarifications

- Q: How many H2H meetings to include? → Last 10 (configurable); see research.md.
- Q: Which league table to use? → Match’s competition/league from fixture data.
- Q: How many news items? → Current behaviour (per-team + matchup); document in data-model.
- Q: How many debates per match? → Configurable (e.g. 3–7); see Epic A1 in user-stories.md.
- Q: Safety classification and context_quality? → Stored for moderation and observability; schema extensions as needed (see data-model).

## Non-Goals (Out of Scope for This Spec)

- User-generated debate topics (only AI-generated prompts in scope).
- Real-time debate updates during the match (pre and post only; “live” phase is optional/future).
- Full admin UI implementation (Epic G may be config table + API first).

## References

- [user-stories.md](./user-stories.md) — Full epics, user stories, and acceptance criteria (A–H).
- [data-model.md](./data-model.md) — Context bundle, entities, data flow.
- [plan.md](./plan.md) — Implementation plan and technical context.
- [contracts/api.yaml](./contracts/api.yaml) — API contract for debate endpoints.
