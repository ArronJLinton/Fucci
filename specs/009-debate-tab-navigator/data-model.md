# Data Model: Debate Tab & Feed (009)

## Overview

No new tables are strictly required if “completion” is derived from existing **votes** (card-level) and **debates** / **debate_cards**. Optional: materialized counters later for scale.

**News / headlines (product)**: Debates surfaced on the main tab are expected to be **generated** using **world football news and headline** context per [004-ai-debate-generator](../004-ai-debate-generator/spec.md) (context bundle → news articles). **009** reads existing `debates` rows; it does not ingest news. Optional future columns (e.g. `source_headline`, `source_url`, `source_published_at`) could be added in **004** or a migration to expose provenance in **DebateSummary**; until then, `headline` / `description` carry the user-visible prompt.

## Existing Entities (reference)

| Entity | Source | Notes |
|--------|--------|--------|
| `debates` | 004/006 | `id`, `match_id`, `headline`, `description`, `debate_type`, … |
| `debate_cards` | 004/006 | Typically three rows per debate (agree / disagree / wildcard); **feed bucketing** uses **any** vote on **agree**/**disagree** (one vote per debate) |
| `votes` | 006 | `user_id`, `debate_card_id`, `vote_type` (`upvote` \| `downvote`) for **card** swipe |
| `comments` | 006 | Threaded replies, `parent_comment_id` for one level |
| `comment_votes` | 006 | Up/down on comments |

## Derived: Debate completion (per user)

For a given `user_id` and `debate_id`:

- Let `V` = whether the user has **any** swipe vote (`vote_type` in `upvote`, `downvote`) on **at least one** `debate_cards` row with `stance IN ('agree', 'disagree')`.
- **Completed** (for **authenticated feed** bucketing) iff `V` is true — **one vote per debate**. Wildcard stance cards are **not** counted toward bucketing.
- **New** iff the user has **no** such vote yet.

## API DTOs (logical)

### DebateFeedResponse

```json
{
  "new_debates": [DebateSummary],
  "voted_debates": [DebateSummary]
}
```

### PublicDebateFeedResponse (guest browse)

```json
{
  "debates": [DebateSummary]
}
```

- **Sort:** `engagement_score` **desc**, tie-break `created_at` **desc** (see `contracts/debates-feed.yaml`).

### DebateSummary (minimal for lists)

- `id`, `match_id`, `headline`, `description` (truncated optional)
- `debate_type`, `created_at`
- `analytics`: optional `total_votes`, `engagement_score` (from `debate_analytics`)
- `card_vote_totals` or aggregate **yes/no** bar data if precomputed for list (optional v1: omit; show placeholder)
- **Optional (provenance, when DB/API extended)**: `source_headline`, `source_url`, `source_published_at` — tie list rows to the **news headline** that grounded generation (004); omit in v1 if not stored. The same fields on **`GET /debates/{id}`** (full debate) drive optional provenance on **detail** (spec FR-009).

### Voted row extras (optional)

- `last_voted_at`: max(`votes.created_at`) for user on any card in debate — for sorting “My Activity”.

## Validation Rules

- **User feed** (`new_debates` / `voted_debates`) **requires** authenticated user (JWT). **Public feed** does not.
- Debate **detail** data for **guests** is read-only (same screen; mutating routes require JWT per 006).
- `limit` query params capped (e.g. 20 per bucket, public feed global limit) to protect DB. **009 v1:** no cursor pagination — single page per request.

## State Transitions

```
[New section] user completes votes on **both** agree and disagree cards → debate moves from new_debates to voted_debates on next feed refresh
```
