# Quickstart: User Engagement for AI Powered Debates

**Feature**: 006-user-engagement-debates  
**Date**: 2026-02-15  
**Target**: Developers testing debate comments, votes, reactions, auth gate, and swipe card voting  
**See also**: [spec.md](./spec.md), [plan.md](./plan.md), [contracts/api.yaml](./contracts/api.yaml)

## Prerequisites

- API and mobile app running (004 debate generator, 005 auth). **Mobile only** (web not available).
- **System user (Fucci)** must exist for seeded comments; the one-time migration at deploy (see Database) creates this user if missing. Authenticated user for testing write actions (reply, vote, react).
- Optional: unauthenticated session to test auth gate modal and to confirm vote/reaction counts are visible without auth.

## Environment

- **API**: Same as 004/005 (`JWT_SECRET`, `DB_URL`, Redis if used). New tables: `comment_votes`, `comment_reactions`; `comments.seeded` column.
- **Mobile**: API base URL and auth token (for reply, vote, react).

## Run the API

From repo root:

```bash
cd services/api && go run ./cmd/api
```

Apply migrations from repo root (adds `seeded` to comments, creates `comment_votes` and `comment_reactions`, ensures system user exists):

```bash
# From repo root (runs migrations in services/api)
yarn migrate
```

Ensure the **system user** (Fucci) exists; migrations or a one-time seed should create it (see [data-model.md](./data-model.md)). Seeded comments are attributed to this user.

## Run the Mobile App

```bash
cd apps/mobile && npx expo start
```

Open a match → Debates tab → open a debate.

- **Header**: Team badge(s) and match score (or "VS" when pre-match).
- **Live debate meter (Debate Pulse)**: At top, shows per-card yes percentages (based on upvotes only); updates live when you vote.
- **Swipe cards**: Stacked cards; swipe right (yes) / left (no) to vote; after three votes the stack hides.
- **Auth gate**: As unauthenticated user, try Reply, Vote, React, or swipe to vote → "Join the conversation" / "Login to vote" modal. Tap **Log in** or **Create account**; after auth you return to the same debate and (best-effort) reply box or reaction picker may auto-open.

## Test Flows

### 1. Swipe card voting (auth required for submit)

- Open a debate with three cards (agree, disagree, wildcard). Confirm cards are **stacked** with the top card active.
- **Swipe right** on the top card: confirm **thumbs up** overlay; card moves away and next card is on top. **Swipe left**: confirm **thumbs down** overlay.
- Confirm **live debate meter** at top updates (e.g. yes/no counts or distribution). Confirm **header** shows team badge(s) and match score.
- As unauthenticated user, attempt swipe: confirm **auth gate** modal; after login, return to debate and vote.
- API: `PUT /api/debates/{debate_id}/cards/{card_id}/vote` with `{ "vote_type": "upvote" }` (yes) or `"downvote"` (no).

### 2. View comments (no auth)

- Open a debate that has seeded comments (and optionally user comments). As unauthenticated user, confirm you can **see vote counts and reaction counts** (read-only; cannot engage).
- Confirm headline, description, and three starter comments with author avatar/name and timestamp (no stance labels — they appear as regular comments, attributed to system user Fucci).
- Confirm subcomments and net score / reactions visible.

### 3. Reply (auth required)

- As authenticated user, tap Reply on a comment, enter text, submit.
- Expect 201 and new comment/subcomment in list.
- As unauthenticated user, tap Reply → auth gate modal; after login, return to debate.

### 4. Upvote / downvote (auth required)

- As authenticated user, tap upvote on a comment; confirm net score updates; tap again to toggle off.
- As unauthenticated user, tap vote → auth gate modal.

### 5. Emoji reaction (auth required)

- As authenticated user, open reaction picker, add emoji; confirm count; add same emoji again to toggle off.
- As unauthenticated user, tap reaction → auth gate modal.

### 6. Seeded comments

- Trigger debate generation (or use existing debate with seeded comments). Ensure system user (Fucci) exists.
- Confirm three seeded comments appear as normal comments (no “AI” or stance labels); attributed to system user (Fucci). No max on emoji reaction types per comment.

## API Endpoints (relative to API base URL)

Paths match the backend router mount (no `/api` prefix unless your server adds it).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /debates/{id} | No | Get debate with cards and card_vote_totals (for meter) |
| GET | /debates/{id}/comments | No | List comments with subcomments, net_score, reactions |
| POST | /debates/{id}/comments | Yes | Create comment or subcomment (body: content, parent_comment_id?); 500 char max; rate limit (e.g. 10/min) → 429 |
| PUT | /debates/{id}/cards/{card_id}/vote | Yes | Swipe vote: body vote_type upvote (yes) or downvote (no); one per user per card |
| PUT | /comments/{id}/vote | Yes | Set comment vote (body: vote_type: upvote \| downvote \| null) |
| POST | /comments/{id}/reactions | Yes | Add/toggle emoji (body: emoji) |
| DELETE | /comments/{id}/reactions?emoji= | Yes | Remove reaction |

## Database

- **users**: Ensure a **system user** (Fucci) exists; create if not (used as author of all seeded comments).
- **comments**: New column `seeded` (boolean, default false). Seeded comments use user_id = system user; no stance stored or exposed in API/UI.
- **comment_votes**: comment_id, user_id, vote_type (upvote/downvote), UNIQUE(comment_id, user_id).
- **comment_reactions**: comment_id, user_id, emoji, UNIQUE(comment_id, user_id, emoji). No max on distinct emoji types per comment.

## Validation (T037)

Run through the test flows above in order:

1. **Swipe card voting** — Stack, swipe right/left, meter update, auth gate when logged out, return-to-debate after login.
2. **View comments** — Headline, three seeded comments (Fucci), net_score and reactions visible without auth.
3. **Reply** — Post comment/reply as authenticated user; 201 and list update; auth gate when logged out, return-to-debate.
4. **Upvote/downvote** — Toggle vote, net score updates; auth gate when logged out.
5. **Emoji reaction** — Add/toggle emoji, count updates; auth gate when logged out.
6. **Seeded comments** — Generate or open debate; confirm three starter comments with no stance labels.

If anything fails: check API base URL and JWT in the mobile app; ensure migrations are applied and system user exists; confirm backend returns the shapes in [contracts/api.yaml](./contracts/api.yaml).
