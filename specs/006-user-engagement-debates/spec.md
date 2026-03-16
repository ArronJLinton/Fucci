# Feature Specification: User Engagement for AI Powered Debates

**Feature Branch**: `006-user-engagement-debates`  
**Created**: 2026-02-15  
**Status**: Draft  
**Input**: Three feature enhancements to the AI Powered Debates experience: headline-driven debate structure with seeded comments, full comment interaction system (replies, votes, emoji reactions), and authentication gate for unauthenticated users.

## Clarifications

### Session 2026-02-15

- Q: When/how is the system user (Fucci) created? → A: Create in a one-time DB migration/seed at deploy; migration ensures Fucci exists before any debate generation.
- Q: Max length for user-written comments and subcomments? → A: 500 characters.
- Q: Require loading and error states for comment list and write actions? → A: Yes — require loading and error (with retry where appropriate) for comment list and for reply/vote/reaction.
- Q: Rate-limit comment creation or vote/reaction? → A: Yes — rate-limit comment creation only (e.g. N per minute per user); votes/reactions not rate-limited for this release.
- Q: After auth, must auto-init (reply focus / reaction picker) be guaranteed? → A: Best-effort — attempt auto-init when return params are present; if state is lost or init fails, show the debate only. No guarantee.
- Q: What does the live debate meter show? → A: One bar for the whole debate (total yes vs no across all 3 cards), with optional per-card breakdown on tap or in a secondary view.
- Q: After user has voted on all three cards, what should the UI show? → A: Hide the card stack; show only the live meter, headline, and comments (stack is for voting only).
- Q: Can the user change a card vote after swiping? → A: Vote is final once submitted; no in-app way to change a card vote.
- Q: When match has no score (pre-match / not started), what should the header show for score? → A: Hide the score area; show only team badges (and "VS" if desired).
- Q: Should card vote (swipe) submission be rate-limited? → A: No — do not rate-limit card votes for this release.

## Relationship to Other Specs

This spec **extends** [004-ai-debate-generator](../004-ai-debate-generator/spec.md). It changes the debate format from a binary two-sided discussion to a headline + description + three AI-seeded comments (agree, disagree, wildcard), and adds comment-level interactions (replies, upvotes/downvotes, emoji reactions) plus an auth gate modal. It depends on [005-user-registration-settings](../005-user-registration-settings/spec.md) for login/registration flows used by the auth gate.

## Scope

1. **Feature 1 — Match Debate Structure: Headline + Seeded Comments**  
   Replace two-sided argument with: headline topic, short description, and three AI-generated starter comments (agree, disagree, wildcard) attributed to the **system user** (Fucci). Stored as standard Comment records with a `seeded` flag; no stance labels in UI or API — they appear identical to other comments. The system user (Fucci) is ensured by a one-time DB migration/seed at deploy.

2. **Feature 2 — Comment Interaction System**  
   Replies and subcomments (one level), upvotes/downvotes on comments, emoji reactions on comments; auth required for all write interactions.

3. **Feature 3 — Authentication Gate Modal**  
   When an unauthenticated user attempts reply, vote, or reaction, show a modal prompting log in or create account; return to the same debate after auth.

4. **Feature 4 — Swipe Card Voting**  
   Users vote on the three debate cards (agree, disagree, wildcard) by swiping right (yes) or left (no) on a stacked card UI, with overlay feedback (thumbs up/down). A live debate meter at the top and team badge + score in the header provide context.

## Feature 4 — Swipe Card Voting (Re-added)

Users vote on the **three debate cards** (agree, disagree, wildcard) via a **swipe gesture** similar to dating apps: **swipe right = yes**, **swipe left = no**. This is separate from comment-level upvotes/downvotes; it is a stance vote on each of the three starter cards.

### UX Requirements

- **Card stack**: The three cards are **stacked on top of one another**, layered so the user can see additional cards behind the top card. Only the top card is fully interactive for swiping.
- **Swipe gestures**: User swipes **right** for yes (agree with this card’s stance) or **left** for no (disagree). On swipe, show an **overlay** with a **thumbs up** (yes) or **thumbs down** (no) emoji for clear feedback.
- **Flow**: User votes on the top card (swipe left/right), then the next card comes to the front; repeat for all three cards (agree, disagree, wildcard). **After the user has voted on all three cards**, hide the card stack and show only the live meter, headline, and comments (stack is for voting only).
- **Live debate meter**: A **live debate meter** at the **top** of the screen shows **one bar for the whole debate** (total yes vs no across all three cards), updating as users vote. An **optional per-card breakdown** (yes/no per stance) is available on tap or in a secondary view.
- **Header**: The **top header** includes **team badge(s)** and, when available, **match score** (e.g. home vs away team logos and current or final score). When the match has no score yet (pre-match or not started), hide the score area and show only team badges (and "VS" if desired).

### Data and API

- One vote per user per **debate_card**: yes (swipe right) or no (swipe left). Reuse or extend existing card-level `votes` storage (e.g. vote_type `yes`/`no` or map to existing upvote/downvote). **Vote is final once submitted**; there is no in-app way to change a card vote.
- API: Submit card vote (authenticated); GET debate or cards with aggregate counts for the live meter. Card vote submission is **not rate-limited** for this release.
- Unauthenticated users can see the meter and cards but must authenticate to submit a swipe vote (auth gate).

### Out of Scope (This Release)

- Editing or deleting seeded comments
- Reporting or flagging comments
- Sorting or filtering comments (by top, newest, etc.)
- Push notifications for replies
- Multi-level nested subcomments (replies to replies)
- AI moderation of user-submitted comments

---

## Feature 1 — Match Debate Structure: Headline + Seeded Comments

### Current Behavior

Each match generates a two-sided debate: one AI argument per side. The experience is binary and passive — users read two positions with no clear entry point for nuance.

### Proposed Behavior

For each match, the system generates:

- **Headline Topic** — A short, punchy, provocative title (e.g., "Is advanced analytics ruining the beauty of the game?")
- **Description** — A 2–3 sentence neutral framing that gives context without taking a side
- **3 Seeded Comments** — AI-generated starter comments (agree, disagree, wildcard perspectives) attributed to the **system user** (Fucci). They are stored and rendered as **regular comments** — no stance labels or tags; they appear identical to the other two and to user-created comments.

Seeded comments look and feel like real user comments to lower the barrier for others to join.

### AI Generation Requirements

| Field | Description | Max Length |
|-------|-------------|------------|
| Headline Topic | Provocative, specific to the match | 80 characters |
| Description | Neutral, informative framing | 300 characters |
| Agree Comment | First-person, opinionated, conversational | 200 characters |
| Disagree Comment | First-person, contrarian, grounded in reasoning | 200 characters |
| Wildcard Comment | Unexpected angle; may reframe or introduce new dimension | 200 characters |

### Comment Attribution

All three seeded comments are attributed to the **system user** (Fucci). The `user_id` on the comment record is the system user. The system user is created in a **one-time DB migration/seed at deploy** so Fucci exists before any debate generation runs. They appear as real authored posts — not labeled as "AI," "Bot," or by stance (agree/disagree/wildcard). For moderation and liability, **Fucci is the attributed author** and is tracked as the author of seeded comments.

### Data Model Notes

- Seeded comments are stored as standard **Comment** records with `seeded: true`, `user_id` = system user (Fucci). No stance field is exposed in the API or UI; they are stored and rendered as regular comments.
- The `seeded` flag is **not** surfaced in the UI.
- Seeded comments participate in the same upvote/downvote, reply, and emoji reaction interactions as user-created comments.

---

## Feature 2 — Comment Interaction System

Users can engage with debate comments through: **replies/subcomments**, **upvotes/downvotes**, and **emoji reactions**.

### 2a — Replies and Subcomments

- Any top-level comment can receive subcomments (threaded replies).
- Subcomments are **one level deep only** — subcomments cannot have their own subcomments.
- **User comments and subcomments**: maximum **500 characters** each; API and UI must validate and enforce this limit.
- Reply UI is triggered by a "Reply" action on any comment.
- Subcomments display: replying user's avatar, username, timestamp, comment text.
- When there are more than 3 replies, the thread is collapsed by default with a "View N more replies" affordance to expand.

### 2b — Upvotes and Downvotes

- Each comment and subcomment has upvote and downvote controls.
- A user may **upvote OR downvote** a comment — not both at once.
- Selecting the same vote again **toggles it off** (deselects).
- Vote counts are displayed as **net score** (upvotes minus downvotes).
- Votes are persisted per user per comment.

### 2c — Emoji Reactions

- Users can add an emoji reaction to any comment or subcomment.
- An emoji picker is shown on tap/click of a reaction affordance.
- Multiple users reacting with the same emoji **increment** that emoji's count.
- A user may add **one reaction per emoji per comment** (toggling removes it).
- There is **no maximum** on the number of distinct emoji reaction types per comment.
- Reactions are displayed as a row of emoji + count below the comment body.

### Interaction States

| Interaction | Authenticated | Unauthenticated |
|-------------|---------------|-----------------|
| View comments | Allowed | Allowed |
| View vote counts and reaction counts | Allowed | Allowed (read-only; cannot engage) |
| Reply / Subcomment | Allowed | Auth gate modal |
| Upvote / Downvote | Allowed | Auth gate modal |
| Emoji reaction | Allowed | Auth gate modal |

**Loading and error states**: The UI MUST show loading states for the comment list and for write actions (reply, vote, reaction). On failure, show a user-friendly error and offer retry where appropriate (e.g. "Couldn't load comments" with Retry; failed vote/reaction with optional retry).

**Rate limiting**: The API MUST rate-limit **comment creation** per user (e.g. N comments per minute); exact limit is implementation-defined. Vote and reaction endpoints are not rate-limited for this release.

---

## Feature 3 — Authentication Gate Modal

### Trigger

When an **unauthenticated** user attempts any write interaction — reply, vote, or emoji reaction — a modal is displayed that blocks the action and prompts authentication.

### Modal Content

| Element | Content |
|---------|---------|
| Title | "Join the conversation" |
| Body | "Log in or create a free account to reply, vote, and react." |
| Primary CTA | "Log in" |
| Secondary CTA | "Create account" |
| Dismiss | X button or tap outside modal |

### Modal Behavior

- Modal appears as an overlay, centered, with darkened backdrop.
- Dismissing returns the user to the debate view; no action is taken.
- "Log in" routes to the login screen; after auth, user is returned to the **same debate**.
- "Create account" routes to the registration screen; after auth, user is returned to the **same debate**.
- The modal is shown **once per session per interaction type** — if the user dismisses and tries again, the modal re-appears.

### Return State

This app is **mobile only** (web not available). After successful authentication, the user is deep-linked back to the specific debate on mobile. **Best-effort auto-init:** when return params (e.g. pendingAction) are present, the app SHOULD attempt to auto-initiate the blocked action (e.g. focus reply input or open reaction picker). If state is lost or auto-init fails, showing the debate alone is acceptable; no guarantee of auto-init is required.

---

## User Stories

### Match Debate Structure

- **As a user** viewing a match, I want to see a headline topic, a short description, and three starter comments (agree, disagree, wildcard) so that the debate feels alive and easy to jump into. The three seeded comments are not labeled by stance; they appear as regular comments.
- The three seeded comments are attributed to the **system user** (Fucci) so the debate has a consistent, trackable author for moderation and liability.

### Comment Interactions

- **As a registered user**, I want to reply to any comment so that I can engage directly with a specific point.
- **As a registered user**, I want to upvote or downvote comments so that I can signal which perspectives I find most compelling or least convincing.
- **As a registered user**, I want to react to comments with emojis so that I can express nuanced reactions beyond agreement or disagreement.
- **As a registered user**, I want to leave subcomments on existing replies so that threaded conversations can develop within the debate.

### Authentication Gate

- **As an unauthenticated visitor**, I want to read all debate content and **see vote counts and reaction counts** so that I can explore the feature before committing to an account; I cannot engage (reply, vote, react) until I am authenticated.
- **As an unauthenticated visitor**, when I try to interact with a comment, I want to see a clear prompt to log in or register so that I understand what I need to do to participate.
- **As a returning user**, after logging in from the auth gate (on mobile), I want to be taken back to the debate I was viewing so I don't lose my place.

### Swipe Card Voting

- **As a user** viewing a debate, I want to see the three stance cards stacked so I can swipe right (yes) or left (no) on each, with a thumbs up/down overlay, so that voting feels quick and engaging like a dating app.
- **As a user**, I want to see a live debate meter at the top and team badge + score in the header so I have match context and aggregate sentiment at a glance.
- **As an unauthenticated visitor**, I want to see the cards and the debate meter but when I try to swipe to vote I am prompted to log in or register (auth gate).
