# Research: Debate Tab & Main Debates Feed (009)

## 1. Feed API: split “new” vs “voted”

**Decision**: Add authenticated **`GET /v1/api/debates/feed`** (or `GET /users/me/debates/feed`) returning two arrays: `new_debates` and `voted_debates`, each item a **DebateSummary** (minimal fields + analytics hints) sufficient for list rows; detail still uses `GET /debates/{id}`.

**Rationale**: `GET /debates/top` is global engagement-sorted and **does not** encode per-user completion. Client-side filtering would require N+1 calls to determine card-vote completion per debate. Server-side join on `votes` (user + debate cards) is one round-trip and consistent.

**Alternatives considered**:

- Client-only: fetch `GET /debates/top?limit=50` + per-debate `getDebate` — rejected (latency, complexity).
- Single list with `user_has_completed_votes` boolean — rejected for UX (product wants two sections without client sort).

## 2. Definition of “voted” / “new”

**Decision**: **Completed** means the user has a card vote recorded for **each of the three** debate cards for that debate (same model as 006 swipe completion).

**Rationale**: Matches existing product rule from 006 (“vote on all three cards”).

**Alternatives considered**:

- Any single card vote → “voted” — rejected (would move debate to activity too early).

## 3. Swipe implementation (React Native)

**Decision**: Use **`react-native-gesture-handler`** + **`react-native-reanimated`** (Expo SDK 54 compatible) for stack top-card pan gestures; overlay labels (Agree / Disagree) per design.

**Rationale**: Project already aligns with Expo; RNGH is the standard for Tinder-style gestures; integrates with Navigation.

**Alternatives considered**:

- `PanResponder` only — acceptable MVP but worse performance and no worklets.
- Third-party card deck libs — only if internal prototype is too slow (defer).

## 4. Navigation structure

**Decision**: Add **Debates** tab mounting a **stack** (`DebatesStack`): root = `MainDebatesScreen`, push = `SingleDebate` (reuse existing screen) with params `{ match, debate }`.

**Rationale**: Matches existing `Home` → `MatchDetails` → `SingleDebate` pattern; minimizes duplicate debate UI.

**Alternatives considered**:

- Modal for detail — rejected (design shows full screen; stack is consistent).

## 5. Design system alignment

**Decision**: Dark theme, lime/red accents as in mocks; implement with **theme tokens** (or existing app palette) so future light mode is possible; ensure **touch targets** ≥ 44pt for swipes and list rows (WCAG).

**Rationale**: Constitution UX + accessibility.

## 6. Backend stack note

**Decision**: API implemented in **Go** with **chi** + **sqlc** (matches repo; constitution mentions Gin — **project uses chi**).

**Rationale**: No new web framework; extend `debates.sql` + `debates.go`.

## 7. Main debates sourced from world football news and headlines

**Decision**: Debates shown on the **main Debates** screen should reflect **top / current world football (soccer) news and headlines** as narrative context. **009** implements **feed APIs and mobile UI only**; it does **not** implement a new news crawler. **Provenance** is ensured by the **[004-ai-debate-generator](../004-ai-debate-generator/spec.md)** pipeline: the generator’s **required** context sources already include **news articles**; jobs and on-demand generation should prioritize **recent, relevant football headlines** (global/world game) when building the context bundle so AI-created debates are timely and recognizable to fans. **Public feed ordering** (engagement-first) naturally surfaces hot, news-adjacent threads once those debates exist in `debates`.

**Rationale**: Single ownership of “what text becomes a debate” (004 + existing news integration); avoids duplicate RSS/API integrations in 009; aligns with product ask without scope explosion.

**Alternatives considered**:

- **009-specific headline scraper** — rejected (duplicates 004 Epic B / news stack; harder to moderate).
- **Client-only headline display without 004 change** — rejected (headlines would not match debate content).
- **Optional `source_headline` / URL on `debates`** — acceptable follow-up (004 or migration); feed DTO may expose optional provenance fields when present (see [data-model.md](./data-model.md)).
