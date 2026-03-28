# Feature Specification: Debate Tab & Main Debates Experience

**Feature Branch**: `009-debate-tab-navigator`  
**Created**: 2026-03-28  
**Status**: Draft  
**Input**: Add a Debates tab to the bottom tab navigator; main screen lists debates not yet voted on (top) and debates already voted on (bottom); swipe UX for agree/disagree; navigation to debate detail for voted items; detail supports comments, subcomments, upvote/downvote per existing engagement spec.

**Design references** (visual targets):

- Main debates: dark theme, “NEW DEBATES” hero card with swipe Agree/Disagree, “MY ACTIVITY” list with consensus bars — see assets under project `assets/` (Velocity Strike–style mock).
- Debate detail: headline, fan consensus, primary vote CTAs, live comments with threading — see Stadium Pulse–style mock (**no** AI analysis strip in 009).

## Relationship to Other Specs

- **Extends / depends on** [006-user-engagement-debates](../006-user-engagement-debates/spec.md): card swipe voting (right = yes/agree, left = no/disagree), comment replies (one level), comment upvotes/downvotes, auth gate for writes. **Guests** may **read** debate lists **and** open **debate detail** read-only (headline, meter, comments); **writes** stay 006 + auth.
- **Depends on** [004-ai-debate-generator](../004-ai-debate-generator/spec.md): debate entity, cards, generation.
- **Depends on** [005-user-registration-settings](../005-user-registration-settings/spec.md): auth for voting and comments.

## Clarifications

### Session 2026-03-28

- Q: What does “not voted yet” mean for a debate? → A: User has **not completed** card swipe voting for that debate (i.e. has not submitted votes for **all three** debate cards). Partial progress stays in “new” until all three cards are voted.
- Q: Swipe direction vs 006? → A: **Swipe right = agree (yes / upvote on card)**, **swipe left = disagree (no / downvote on card)** — aligned with 006 Feature 4 and existing `PUT .../cards/{cardId}/vote` semantics.
- Q: “Aggregated debates” on main screen? → A: **Server-provided feed** split into `new_debates` and `voted_debates` (see plan/contracts); client renders top section then bottom section. Ordering within each list: **new** by engagement/recency as specified in API; **voted** by last activity or vote time (API default).
- Q: Can guests open the Debates tab without logging in? → A: **Yes — public read-only** endpoint(s) let anyone **browse** debate summaries for the main screen. **Engagement requires auth:** card swipe vote, comment, reply, subcomment, and comment upvote/downvote **require authentication** (auth gate per 006). **Authenticated** feed returns the per-user **`new_debates`** / **`voted_debates`** split; **guest** UI uses the public payload (no user-specific “voted” history).
- Q: Can guests open **debate detail**? → A: **Yes — full** detail **read-only**: headline, fan consensus / meter, and **comments** (threading) are **visible**; **card swipe vote, new comment, reply, and comment upvote/downvote** use the **auth gate** (006).
- Q: How should **My Activity** appear for **guests** on the main screen? → A: Section **remains visible** with an **empty state** and a **sign-in CTA** (not hidden; not copy-only without CTA).
- Q: How should **`GET …/debates/public-feed`** order the **`debates`** list? → A: **Engagement-first** (e.g. analytics / engagement score **desc**), **tie-break** **`created_at` desc** — document exact fields in `contracts/`.
- Q: **AI analysis strip** on debate detail? → A: **Removed from 009** — do **not** implement or show an AI analysis strip on debate detail (headline, consensus, vote CTAs, comments only).
- Q: Feed **pagination** in v1? → A: **Capped lists only** — `limit` query params per route; **no** cursor / `next_page` in **009 v1** (revisit in a later release if lists grow).

## User Scenarios & Testing

### User Story 1 — Open Debates from bottom tabs (Priority: P1)

As a user, I tap **Debates** in the bottom tab bar and land on the **main debates** screen so I can browse debates in one place.

**Why this priority**: Core navigation entry for the feature.

**Independent Test**: With tabs configured, tapping Debates shows the main debates layout (sections + lists) without requiring match navigation.

**Acceptance Scenarios**:

1. **Given** the app shows the main tab navigator, **When** the user taps the Debates tab, **Then** the main debates screen is shown.
2. **Given** the user is **authenticated**, **When** the screen loads, **Then** the client fetches the **authenticated** feed (`new_debates` / `voted_debates`) and shows loading then content or error.
3. **Given** the user is **not authenticated**, **When** the screen loads, **Then** the client fetches the **public read-only** feed and shows browseable content without exposing another user’s data; **My Activity** shows **empty state + sign-in CTA** (no personalized rows until sign-in).

---

### User Story 2 — New vs voted sections (Priority: P1)

As a user, I see debates I still need to vote on **above** debates I already participated in.

**Why this priority**: Primary information architecture from product input.

**Independent Test**: With mock API returning both buckets, UI order is “new” block first, “my activity” second.

**Acceptance Scenarios**:

1. **Given** the feed contains both unvoted and completed debates, **When** the main screen renders, **Then** the **new/unvoted** block appears **above** the **voted / my activity** block.
2. **Given** one section is empty, **When** the screen renders, **Then** the empty section shows an appropriate empty state (copy + optional CTA) without breaking layout.

---

### User Story 3 — Swipe to vote on top debate card (Priority: P1)

As a user, I use **dating-app-style** swipes on the featured debate: **swipe right = agree**, **swipe left = disagree**, submitting the corresponding card vote.

**Why this priority**: Differentiator UX; maps to card vote API.

**Independent Test**: Swipe gestures trigger authenticated `PUT /debates/{id}/cards/{cardId}/vote` with `vote_type` upvote/downvote; optimistic UI optional; errors surfaced.

**Acceptance Scenarios**:

1. **Given** a debate with cards loaded and user authenticated, **When** the user swipes the top card **right**, **Then** an **agree/yes** vote is recorded for that card (upvote).
2. **Given** the same, **When** the user swipes **left**, **Then** a **disagree/no** vote is recorded (downvote).
3. **Given** the user is **not** authenticated, **When** they attempt to swipe-vote, **Then** the **auth gate** flow applies per 006 (modal → login/signup → return).

---

### User Story 4 — Open debate detail from My Activity (Priority: P2)

As a user, I open the **debate detail** screen from **My Activity** (signed-in) or from the **public browse list** (guest).

**Why this priority**: Secondary navigation; reuses Single Debate / detail stack.

**Acceptance Scenarios**:

1. **Given** a row in the voted list, **When** the user taps it, **Then** navigation opens the debate detail screen with the correct `debateId` (and match context as needed).
2. **Given** deep link or refresh, **When** debate loads, **Then** headline, meter, and comments load per existing APIs.
3. **Given** a **guest** taps a debate from the public browse list, **When** detail opens, **Then** headline, meter, and **comments** are shown **read-only**; actions that **mutate** state (card vote, comment, reply, comment vote) trigger the **auth gate**.

---

### User Story 5 — Comment, reply, comment votes on detail (Priority: P1)

As a user on debate detail, I can **comment**, **reply** (one level), and **upvote/downvote** comments per 006.

**Why this priority**: Required engagement; largely backend-complete.

**Acceptance Scenarios**:

1. **Given** an authenticated user on debate detail, **When** they post a comment, **Then** the comment appears per POST comments API.
2. **Given** a comment, **When** they reply, **Then** a subcomment is created (one level max).
3. **Given** a comment, **When** they upvote or downvote, **Then** vote totals update per comment vote API.
4. **Given** a **guest** on debate detail, **When** they read the thread, **Then** comments display read-only; **Given** they attempt to post, reply, or vote on a comment, **Then** the auth gate applies.

---

### Edge Cases

- **Guest** → public read-only feed only; **no** personalized `voted_debates`; **My Activity** block shows **empty state + sign-in CTA**; **debate detail** is read-only (headline, meter, comments visible); engagement actions show auth gate.
- User has **no** debates in either list → full empty state for main screen (authenticated); guests may still see public trending rows.
- **Partial** card votes (1–2 cards only) → debate stays in **new** until all three cards voted (authenticated feed only).
- **Rate limits / offline** → show error and retry per 006 for comments; card votes not rate-limited per 006.
- **Long lists** → **v1:** capped via `limit` only (**no** cursor pagination in 009); optional infinite scroll UX is client-side over the capped window only.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Expose a **Debates** tab in the bottom tab navigator (Expo / React Navigation).
- **FR-002**: Provide a **public read-only** HTTP API to load debate summaries for **guests** (browse-only; no per-user vote history; document sort keys in `contracts/`). **Ordering:** engagement-first (e.g. analytics score **desc**), tie-break **`created_at` desc** — stable, testable ordering.
- **FR-002b**: When **authenticated**, main debates screen calls the **user feed** API that returns **`new_debates`** and **`voted_debates`** (or equivalent split documented in contracts).
- **FR-002c**: **Vote** (card swipe), **comment**, **reply**, and **comment upvote/downvote** **require authentication**; unauthenticated users get the 006 auth gate, not silent failure.
- **FR-003**: UI places **new** above **voted** for signed-in users; labels aligned with design (“NEW DEBATES” / “MY ACTIVITY” or accessible equivalents). **Guests**: top browse area from public feed; **My Activity** section **stays visible** with **empty state + sign-in CTA** (see contracts for list payload).
- **FR-004**: Featured **top-card** swipe UX: right = agree, left = disagree; integrates with existing card vote endpoint.
- **FR-005**: Tapping a **voted** row navigates to **debate detail** (stack screen) with debate + match params. **Guests** may open the same detail screen from the **public** browse list.
- **FR-006**: Debate detail supports **comments**, **subcomments**, **comment upvote/downvote** using existing debate comment APIs (006).
- **FR-006b**: On debate detail, **guests** **read** headline, consensus/meter, and comment threads; **mutating** actions require authentication (006 gate).
- **FR-006c**: Debate detail **excludes** an **AI analysis strip** (not in 009 scope).
- **FR-007**: Loading and error states for feed and detail (constitution / 006 alignment).
- **FR-008**: **009 v1** feed APIs use **capped** `limit` parameters only; **no** cursor-based pagination in contract or response (future iteration may add).

### Key Entities *(include if feature involves data)*

- **Debate**, **DebateCard**, **Vote** (card-level), **Comment**, **CommentVote** — see [data-model.md](./data-model.md).

### Success Criteria *(mandatory)*

- **SC-001**: User can reach main debates from tabs in ≤ 2 taps from cold start (after app open).
- **SC-002**: P95 feed API response &lt; 200ms on standard queries (constitution; validated in staging).
- **SC-003**: No uncaught errors on swipe when authenticated; **auth gate** when not logged in; guests can load the public read-only feed **and** debate detail read-only content without errors when APIs succeed.

## Assumptions

- Mobile (Expo) is in scope first; web out of scope unless explicitly added later.
- Match-scoped debate entry from Home remains; Debates tab is an additional global entry.
