# AI-Powered Debate Generator — User Stories

**Feature**: 004-ai-debate-generator  
**Aligned with**: [spec.md](./spec.md)

This document holds the full epic and user story set with acceptance criteria. Scope, roles, and definitions are in the main spec.

---

## Epic A — Debate Generation (Core AI)

### A1. Generate debates for a match (scheduled pre-match)

**User Story**  
As a fan, I want to see a set of AI-generated debates for a match before kickoff so I can join discussions early.

**Acceptance Criteria**

- Given a match scheduled within X hours, the system generates N debates (configurable, e.g. 3–7).
- Each debate includes:
  - **title** (short)
  - **prompt** (two-sided framing)
  - **side_a_label**, **side_b_label** (e.g. “Yes / No”, “Team A / Team B”, “Overrated / Elite”)
  - **topic_tags** (e.g. tactics, form, rivalry, player)
  - **source_context_ids** (refs to stat/news items used)
  - **safety_classification** (approved / needs_review / blocked)
- Debates are stored and retrievable via API by `match_id`.
- Debates do not regenerate unnecessarily if already created within a TTL.

---

### A2. Generate debates on demand (fallback)

**User Story**  
As a fan, if debates are missing for a match, I want the system to generate them on demand so the experience never feels empty.

**Acceptance Criteria**

- If debates for `match_id` are missing, API triggers generation job and returns:
  - either a “pending” state with polling, or
  - a minimal default debate set (template-based) immediately.
- On-demand generation is rate-limited per match.
- On-demand generation completes within a target SLA (e.g. &lt; 10–20s) or returns a graceful fallback.

---

### A3. Generate debates during/after match (post-match narratives)

**User Story**  
As a fan, I want debates to update after the match based on outcome and key moments so discussion stays relevant.

**Acceptance Criteria**

- System generates a post-match debate set when match status changes to “FT”.
- Post-match debates reference: result, key stats, standout players, turning points (if available).
- Post-match debates are stored separately or versioned (e.g. phase = pre | live | post).

---

### A4. Debate variety & deduplication

**User Story**  
As a fan, I want debates to feel fresh and non-repetitive across matches so I don’t see the same questions every time.

**Acceptance Criteria**

- Debates are checked against prior prompts for:
  - near-duplicate similarity threshold
  - repeated wording patterns
- If duplicates detected, generator produces alternatives.
- Debates are diversified across tags: tactics, form, rivalry, player spotlight, coaching, predictions.

---

### A5. Debate quality guardrails (structured prompting)

**User Story**  
As the system, I want debate prompts to follow a consistent structure so they’re easy to read and argue with.

**Acceptance Criteria**

- Debates conform to a schema:
  - max title length
  - max prompt length
  - includes both sides with balanced framing
- Prompt avoids:
  - factual uncertainty presented as certainty
  - personal attacks
  - hate/harassment content
- Debates contain a “what this debate is about” one-liner (optional but recommended).

---

## Epic B — Context Building (Data Inputs)

### B1. Build a context bundle for each match

**User Story**  
As the system, I need to compile match context (H2H, form, player stats, injuries, news) so debates are grounded.

**Acceptance Criteria**

- Context bundle includes structured fields (even if some are empty):
  - match metadata (teams, venue, competition, kickoff time)
  - H2H summary (last X meetings)
  - team form (last X matches)
  - key players + stats (goals, assists, ratings if available)
  - injuries/suspensions (if available)
  - recent news headlines + snippets + source IDs
- Bundle is cached with TTL and stored for traceability.

---

### B2. Degrade gracefully when data is missing

**User Story**  
As the system, when some data sources are unavailable, I still want to generate debates using what’s available.

**Acceptance Criteria**

- Generator can produce debates with partial context (e.g. only form + H2H).
- Missing fields do not break generation.
- Debates created under partial data are tagged **context_quality = partial**.

---

### B3. Control what sources can influence debates

**User Story**  
As an admin, I want to configure which sources (news providers, stat providers) are allowed so we manage credibility and licensing.

**Acceptance Criteria**

- Admin can enable/disable sources at runtime (config table or admin UI).
- Debates store which source IDs were used.
- System can regenerate debates if a source becomes disallowed.

---

## Epic C — Delivery in Product (User Experience)

### C1. Show debates on match details screen

**User Story**  
As a fan, I want debates visible on the match page so it’s easy to join the conversation.

**Acceptance Criteria**

- Debates appear in a dedicated “Debates” tab.
- Debates load quickly using cached results.
- Empty state explains if debates are still generating.

---

### C2. Support sorting and “trending” debates

**User Story**  
As a fan, I want to sort debates by “top” or “new” so I can find the best discussions.

**Acceptance Criteria**

- Sorting options: Top, New, Trending.
- Top uses vote score over a time window.
- Trending uses velocity (votes/comments per minute) with decay.

---

### C3. Deep link to a debate thread

**User Story**  
As a fan, I want to share a debate link so friends can jump into the exact thread.

**Acceptance Criteria**

- Each debate has a unique URL/deep link.
- Opening the link routes to match → debates → selected thread.

---

## Epic D — Engagement (Comments + Voting + Ranking)

### D1. Comment on a debate

**User Story**  
As a fan, I want to post a comment under a debate so I can contribute my take.

**Acceptance Criteria**

- Auth required to comment.
- Comments support basic text (MVP).
- Comment appears immediately (optimistic UI allowed).
- Rate limiting prevents spam.

---

### D2. Upvote/downvote comments

**User Story**  
As a fan, I want to upvote/downvote comments so the best arguments rise to the top.

**Acceptance Criteria**

- Each user can cast one vote per comment (toggleable).
- Vote state is persisted and reflected in ranking.
- Score updates quickly (Redis counter acceptable) and reconciles with DB.

---

### D3. Thread ranking logic

**User Story**  
As a fan, I want top comments to show first so reading is valuable.

**Acceptance Criteria**

- Default sorting uses score with time decay (Reddit-like) OR a simpler MVP: score desc, created_at desc.
- Admin can adjust algorithm weights (optional).

---

## Epic E — Safety, Trust, and Moderation

### E1. Moderate AI-generated debate prompts

**User Story**  
As an admin, I want to review or block debates that violate policy so the platform stays safe.

**Acceptance Criteria**

- Each debate is automatically classified: **approved** | **needs_review** | **blocked**.
- Admin can approve/block/edit.
- Blocked debates are not shown to users.

---

### E2. Moderate user comments

**User Story**  
As an admin, I want comment reporting and enforcement so harassment/spam is handled quickly.

**Acceptance Criteria**

- Users can report comments (reason + optional note).
- Admin review queue shows: comment text, user info, debate/match context, report counts.
- Admin actions: delete, warn, suspend/ban.

---

### E3. Abuse prevention (spam & brigading)

**User Story**  
As the system, I want to detect spam/brigading so voting stays fair.

**Acceptance Criteria**

- Rate limits for: comment creation, votes per minute, on-demand debate generation triggers.
- Basic heuristics flag suspicious activity (e.g. repeated text, burst votes).
- Admin can review flagged users.

---

## Epic F — Cost Control & Performance

### F1. Cache generated debates by match + phase

**User Story**  
As the system, I want caching so we don’t pay to regenerate the same debates repeatedly.

**Acceptance Criteria**

- Debates cached using match_id + phase + context_version.
- TTL configurable (e.g. pre-match until kickoff, post-match for days).
- If context changes materially (e.g. late injury news), context_version increments and can trigger refresh.

---

### F2. Rate-limit AI calls

**User Story**  
As the system, I want to prevent runaway AI usage so costs don’t spike.

**Acceptance Criteria**

- Per-match generation limited (e.g. max 3 regenerations/day).
- Per-admin “regenerate” action is permissioned and audited.
- Alerts/logging if thresholds exceeded.

---

### F3. Background job processing for generation

**User Story**  
As the system, I want debate generation to run async so API remains responsive.

**Acceptance Criteria**

- API enqueues job and returns pending status.
- Worker processes job and stores results.
- Observability: job status, duration, failures.

---

## Epic G — Admin Controls & Observability

### G1. Debate generator configuration panel (or config table)

**User Story**  
As an admin, I want to configure debate generation parameters so content fits our brand.

**Acceptance Criteria**

- Configurable values include: debates per match, allowed topics/tags, profanity strictness, max prompt length, pre/live/post generation timing.
- Changes apply without redeploy (where feasible).

---

### G2. Audit trail for edits/regenerations

**User Story**  
As an admin, I want an audit trail so we can track what changed and why.

**Acceptance Criteria**

- Every admin edit, approval, block, regeneration is recorded with: admin_id, timestamp, previous and new values, reason (optional).

---

### G3. Metrics dashboard for debates

**User Story**  
As a product owner/admin, I want to track debate performance so we can iterate on quality.

**Acceptance Criteria**

- Metrics tracked by match and overall: debates generated count, views per debate, comments per debate, votes per comment, report rate, generation time + failure rate.
- Exportable report (CSV) is a plus.

---

## Epic H — Future Enhancements (Optional / Post-MVP)

### H1. Personalized debates per user

**User Story**  
As a fan, I want debates tailored to my favorite teams/players so the feed is more relevant.

**Acceptance Criteria**

- Users can set favorite teams/players.
- Debate feed prioritizes those matches.

---

### H2. Multi-language debate generation

**User Story**  
As a fan, I want debates in my language so engagement is accessible globally.

**Acceptance Criteria**

- Language preference stored per user.
- Debates generated or translated with correct locale.
- Admin can restrict supported languages initially.

---

### H3. “AI Referee” summarization

**User Story**  
As a fan, I want an AI summary of both sides so I can catch up quickly.

**Acceptance Criteria**

- Summaries generated after thread reaches thresholds (e.g. 50 comments).
- Summaries cite top-voted arguments (no hallucinated facts).

---

## MVP Cut Recommendation

For the leanest MVP that still feels complete, prioritise:

- **A1, B1, C1** — Generate + context + display
- **D1, D2** — Comment + vote
- **E1, E2** — Moderation basics
- **F1, F3** — Caching + async worker

Implementation order can follow: context bundle (B1) → generation (A1, A2, A3, A5) → caching (F1) → async (F3) → delivery (C1) → engagement (D1, D2) → moderation (E1, E2).
