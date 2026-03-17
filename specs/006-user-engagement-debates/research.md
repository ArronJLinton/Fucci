# Research: User Engagement for AI Powered Debates

**Feature**: 006-user-engagement-debates  
**Date**: 2026-02-15

## 1. Seeded Comments vs. Debate Cards

**Decision**: Store the three AI-generated starter comments (agree, disagree, wildcard) as **Comment** rows with a `seeded` boolean flag, linked to the **Debate** (not to debate_cards). Keep **debate_cards** for any stance metadata/labels if still needed for UI; the primary “starter” content is the seeded comments.

**Rationale**: Spec states “seeded comments are stored as standard Comment records with seeded: true” and “subject to the same upvote/downvote, reply, and emoji reaction interactions.” Using the existing comments table avoids duplicate structures and keeps one interaction model for all comment-like content.

**Alternatives considered**: (a) Store seeded text only on debate_cards — rejected because comments have their own reply/vote/reaction model and the spec explicitly wants “comments.” (b) New “seeded_comments” table — rejected to avoid schema proliferation; a flag on comments is sufficient.

---

## 2. Comment-Level Votes and Emoji Reactions

**Decision**: Introduce **comment-level** vote and reaction storage. Current schema has `votes` keyed by `debate_card_id`. Add either: (A) a `comment_votes` table (comment_id, user_id, vote_type: upvote|downvote) and a `comment_reactions` table (comment_id, user_id, emoji), or (B) extend a single “comment_engagement” table with type (vote/reaction) and value. Prefer **separate tables** (comment_votes, comment_reactions) for clarity and to match existing votes semantics (one row per user per target per type).

**Rationale**: Spec requires upvote/downvote and emoji reactions on **comments** (and subcomments). Existing `votes` is tied to debate_cards. Adding comment-scoped tables keeps debate-card votes unchanged and allows net score and per-emoji counts with simple aggregates.

**Alternatives considered**: Reusing `votes` with nullable comment_id — possible but mixes card votes and comment votes in one table; separate tables give clearer constraints and indexes.

---

## 3. Subcomment Depth and Thread Collapse

**Decision**: Enforce **one level of nesting only**: comments have `parent_comment_id`; subcomments have a non-null parent_comment_id and must have a parent that is a top-level comment (parent_comment_id IS NULL). API and UI reject replies to a subcomment. “View N more replies” is a UI-only behavior (e.g., collapse when reply count > 3); backend returns all replies for a comment.

**Rationale**: Spec states “subcomments are nested one level deep only — subcomments cannot have their own subcomments.” Existing schema already has parent_comment_id; we add a check (or application logic) that the parent of a new comment is either null (top-level) or has null parent (first-level reply).

**Alternatives considered**: Allowing arbitrary depth — rejected per spec.

---

## 4. Attribution of Seeded Comments to “Fucci”

**Decision**: Store seeded comments with `user_id` set to the **system user** (Fucci). Create a system user (e.g. display name “Fucci”) if one does not exist. Stances (agree, disagree, wildcard) are **not** labeled or tagged in the UI or API — seeded comments are rendered and stored as regular comments. For moderation and liability, **Fucci (the attributed author / system user) is liable**; we keep track of the author via the comment’s user_id.

**Rationale**: Clarification: “The user_id for the comment record will be the system user”; “Fucci is liable so we can keep track of the author.” So a single system user represents Fucci; all seeded comments point to that user for consistent attribution and moderation.

**Alternatives considered**: Using the triggering user as author — rejected; spec now requires system user only. Stance labels in UI — rejected; spec says no labeling, render as regular comments.

---

## 5. Auth Gate Modal and Return State

**Decision**: Auth gate is a **client-side modal** on **mobile only** (web is not available). On “Log in” or “Create account,” navigate to the existing login/registration flow (005); after successful auth, **deep-link back** to the debate (e.g. debate ID in navigation state) and, where feasible, **auto-initiate** the blocked action (e.g. focus reply input or open reaction picker). Use navigation state or a stored “pending action” so the app knows where to return and what to open. Return-to-debate deep link is feasible on mobile given the current auth flow (navigation stack + state).

**Rationale**: Clarification: “This is a mobile app. Web not available.” So return-to-debate and auth gate apply to mobile only; no web deep-link design required.

**Alternatives considered**: Designing for web — out of scope; mobile-only.

---

## 6. Emoji Picker and Reaction Set

**Decision**: There is **no maximum** on the number of emoji reaction types supported per comment. Store emoji as a short code or character in the database; one reaction per user per emoji per comment (toggle removes). The UI may offer a picker with a suggested set for UX; the backend does not limit how many distinct emoji types can appear on a single comment.

**Rationale**: Clarification: “no max on emoji reaction” — so we do not enforce a cap on distinct emoji types per comment.

**Alternatives considered**: Fixed set or max per comment — rejected per clarification.

---

## 7. Unauthenticated Users and Vote/Reaction Counts

**Decision**: **Unauthenticated users can see vote counts and reaction counts** on comments; they cannot engage (reply, vote, or react) until authenticated. The auth gate modal is shown when they attempt any write action.

**Rationale**: Clarification: “Yes. But they can't engage with the post until they are authenticated.” So list comments and counts are visible to all; only write operations are gated.

---

## 8. Swipe Card Voting UX and Data

**Decision**: **Re-add card-level voting** with a **swipe UX**: three cards (agree, disagree, wildcard) stacked; user swipes **right = yes**, **left = no** on each card in order. Show **thumbs up / thumbs down overlay** on swipe. Use existing **`votes`** table keyed by `debate_card_id` and `user_id`; store swipe as `vote_type` **`yes`** or **`no`** (add if not present) or map to existing upvote/downvote for card scope. One vote per user per card; last swipe wins if user re-votes. **Live debate meter** at top shows aggregate (e.g. % yes/no per card or overall). **Header**: team badge(s) + match score from match context.

**Rationale**: User request: “similar to dating app — swipe right yes, swipe left no; vote on 3 cards; stacked cards with overlay; live debate meter at top; team badge and score at top header.” Reusing `votes` avoids new tables; card votes are distinct from comment_votes (comment-level up/down). Auth required to submit card vote; unauthenticated can see meter and cards (read-only) and get auth gate on swipe.

**Alternatives considered**: (a) New `card_votes` table — rejected to reuse existing `votes` with debate_card_id. (b) No meter — rejected; user asked for live debate meter. (c) Swipe only on first card — rejected; user said “vote on 3 cards” so all three get swipe.
