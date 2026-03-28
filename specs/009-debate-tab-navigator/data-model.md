# Data Model: Debate Tab & Feed (009)

## Overview

No new tables are strictly required if “completion” is derived from existing **votes** (card-level) and **debates** / **debate_cards**. Optional: materialized counters later for scale.

## Existing Entities (reference)

| Entity | Source | Notes |
|--------|--------|--------|
| `debates` | 004/006 | `id`, `match_id`, `headline`, `description`, `debate_type`, … |
| `debate_cards` | 004/006 | Three cards per debate (agree / disagree / wildcard stances) |
| `votes` | 006 | `user_id`, `debate_card_id`, `vote_type` (`upvote` \| `downvote`) for **card** swipe |
| `comments` | 006 | Threaded replies, `parent_comment_id` for one level |
| `comment_votes` | 006 | Up/down on comments |

## Derived: Debate completion (per user)

For a given `user_id` and `debate_id`:

- Let `C` = count of distinct `debate_cards` for that debate (expected **3**).
- Let `V` = count of distinct cards for which the user has **any** card vote row in `votes`.
- **Completed** iff `V >= C` (typically `V == 3`).
- **New** iff `V < C` (includes zero votes).

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

### Voted row extras (optional)

- `last_voted_at`: max(`votes.created_at`) for user on any card in debate — for sorting “My Activity”.

## Validation Rules

- **User feed** (`new_debates` / `voted_debates`) **requires** authenticated user (JWT). **Public feed** does not.
- Debate **detail** data for **guests** is read-only (same screen; mutating routes require JWT per 006).
- `limit` query params capped (e.g. 20 per bucket, public feed global limit) to protect DB. **009 v1:** no cursor pagination — single page per request.

## State Transitions

```
[New section] user completes 3rd card vote → debate moves from new_debates to voted_debates on next feed refresh
```
