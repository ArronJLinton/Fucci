# Data Model: User Engagement for AI Powered Debates

**Feature**: 006-user-engagement-debates  
**Date**: 2026-02-15  
**Extends**: [004-ai-debate-generator](../004-ai-debate-generator/data-model.md) (debates, debate_cards, comments), existing `comments` and `votes` schema

## Purpose

This document describes schema and entity changes for: (1) seeded comments on debates, (2) comment-level upvotes/downvotes and emoji reactions, and (3) subcomment rules. It reuses debates and debate_cards where applicable and extends comments and votes/reactions.

## Entity Overview

| Entity / Concept | Action | Purpose |
|------------------|--------|---------|
| debates | Use | Existing; headline + description already present |
| debate_cards | Use | Existing; optional for backward compatibility; seeded comments are not labeled by stance in UI |
| comments | Extend | Add `seeded` flag; seeded comments are created with debate_id, user_id = **system user** (Fucci), content from AI; no stance label in API/UI |
| comment_votes | New | One row per (comment_id, user_id); vote_type in ('upvote','downvote'); net score = SUM(upvote) - SUM(downvote) per comment |
| comment_reactions | New | One row per (comment_id, user_id, emoji); same emoji from same user toggles off |

## Comments (Extended)

**Existing columns** (from 004): `id`, `debate_id`, `parent_comment_id`, `user_id`, `content`, `created_at`, `updated_at`.

**Additions for 006**:

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| seeded | BOOLEAN | NOT NULL DEFAULT false | True for AI-generated starter comments (agree/disagree/wildcard); not shown in UI |

**Validation / rules**:

- Seeded comments: `seeded = true`, `parent_comment_id IS NULL`, `user_id` = **system user** (Fucci). Create a system user if one does not exist. No stance field is exposed in API or UI — stored and rendered as regular comments. Content length per spec (e.g. 200 chars). For moderation/liability, Fucci (system user) is the attributed author.
- Subcomments: `parent_comment_id` set to a comment that has `parent_comment_id IS NULL` (one level only). Reject inserts where parent is already a subcomment.

## Comment Votes (New)

**Purpose**: Store upvote/downvote per user per comment. One row per (comment_id, user_id) with vote_type; or one row per (comment_id, user_id, vote_type) if we allow storing up/down separately. Spec: “A user can upvote OR downvote — not both; selecting same vote toggles off.” So at most one vote row per (comment_id, user_id) with vote_type in ('upvote','downvote').

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| id | SERIAL | PK | |
| comment_id | INTEGER | NOT NULL | FK to comments(id) ON DELETE CASCADE |
| user_id | INTEGER | NOT NULL | FK to users(id) ON DELETE CASCADE |
| vote_type | VARCHAR(10) | NOT NULL | 'upvote' or 'downvote' |
| created_at | TIMESTAMP | DEFAULT now | |
| UNIQUE(comment_id, user_id) | | | One vote per user per comment (up or down); toggle = update to opposite or delete |

**Indexes**: comment_id (list votes for comment), user_id (optional, for “my votes”).

**Net score**: For a comment, net = COUNT(vote_type='upvote') - COUNT(vote_type='downvote'); or store as cached value on comment or analytics table for performance.

## Comment Reactions (New)

**Purpose**: Store emoji reactions per user per comment. Multiple users can add the same emoji (count increments). One row per (comment_id, user_id, emoji). Toggling removes the row.

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| id | SERIAL | PK | |
| comment_id | INTEGER | NOT NULL | FK to comments(id) ON DELETE CASCADE |
| user_id | INTEGER | NOT NULL | FK to users(id) ON DELETE CASCADE |
| emoji | VARCHAR(20) | NOT NULL | Emoji code or character; no max on distinct emoji types per comment |
| created_at | TIMESTAMP | DEFAULT now | |
| UNIQUE(comment_id, user_id, emoji) | | | One reaction per user per emoji per comment |

**Indexes**: comment_id (for listing reactions by comment).

**Display**: Group by emoji, count; show row of emoji + count below comment.

## Seeded Comment Creation Flow

1. Ensure a **system user** (Fucci) exists; create one if not (e.g. display_name “Fucci”).
2. When a debate is generated (or when “generate set” runs), the AI returns headline, description, and three comment texts (agree, disagree, wildcard).
3. For each of the three, insert into `comments`: debate_id, parent_comment_id = NULL, user_id = **system user (Fucci)**, content = AI text, seeded = true. Do **not** store or expose stance (agree/disagree/wildcard) in the API or UI — they are regular comments. Order by creation order / id.

## Relationship to 004

- **Debates / debate_cards**: Unchanged for 006; debate still has headline, description; cards may remain for backward compatibility or be phased out in favor of “seeded comments” as the only starter content.
- **votes (existing)**: Used for **card-level swipe voting** (Feature 4): one vote per user per debate_card; store as `vote_type` **upvote** (swipe right / yes) or **downvote** (swipe left / no), `emoji` NULL. Application enforces one vote per user per card (replace on re-vote). Comment-level voting is in **comment_votes**.
- **comments**: Extended with `seeded`; subcomment rule enforced in application (and optionally with a CHECK or trigger).

## Card Vote (Swipe) Semantics

- **votes** table: `debate_card_id`, `user_id`, `vote_type` in ('upvote', 'downvote') for card swipe; `emoji` NULL. **Upvote** = swipe right (yes), **downvote** = swipe left (no).
- One vote per user per card: on submit, replace any existing row for (debate_card_id, user_id) so the user has at most one card vote per card.
- **Live debate meter**: Aggregate counts per card (yes_count, no_count) or per debate (e.g. total yes vs no across all three cards) from `votes`; return in GET debate or GET debate cards response for the meter UI.

## Summary

| Entity | Action | Notes |
|--------|--------|-------|
| users (system) | Ensure | Create system user (Fucci) if not exists; used as author of all seeded comments |
| comments | Extend | Add `seeded`; seeded comments use user_id = system user; no stance in API/UI |
| comment_votes | New | comment_id, user_id, vote_type (upvote/downvote), UNIQUE(comment_id, user_id) |
| comment_reactions | New | comment_id, user_id, emoji; no max on emoji types per comment; UNIQUE(comment_id, user_id, emoji) |
| votes (debate_cards) | Use | Card swipe: upvote=yes, downvote=no; one per user per card; feed live debate meter |
| Subcomment depth | Logic | Enforce parent is top-level only (parent_comment_id IS NULL) |
