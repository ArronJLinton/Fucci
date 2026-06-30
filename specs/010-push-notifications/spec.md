# Feature Specification: Push Notifications (WC v1)

**Feature Branch**: `feat/push-notifications`  
**Created**: 2026-06-28  
**Status**: Draft — Phase 1 covers **infrastructure only** (no campaign selectors yet)

## Scope phases

| Phase | Deliverable |
|-------|-------------|
| **1 — Infra (this doc’s focus)** | Device registration, preferences, Expo send pipeline, delivery logging, deep links |
| **2 — Campaigns** | Debate 6pm local, news 12pm local, match post-FT (max 2/day) |
| **3 — Post-WC** | Transfer/historic selectors; product-mode switch |

## Functional summary (locked)

See conversation 2026-06-28. Highlights:

- **Debate:** 6pm **local**, daily during WC; top unvoted debate by boosted engagement; title = headline; skip if all voted.
- **News:** 12pm **local**; heuristic rank; exclude opened articles; always send best remaining.
- **Match:** post-FT ~1h; FOX/ESPN/Telemundo Shorts or “Match finished — debates live”; max **2/day**; FIFA top 25 → 50 marquee.
- **Opt-in** required; per-category toggles in v1 infra (defaults off until user enables).

---

## Technical requirements — Phase 1 infrastructure

### 1. Mobile client (Expo)

**Dependencies**

- `expo-notifications` (SDK 54 compatible)
- `expo-device` (physical device check)
- Existing: `expo-constants` (EAS `projectId` for push token)

**Capabilities**

| Capability | Requirement |
|------------|-------------|
| Permission | Request after explicit user action (settings toggle or onboarding sheet), not on cold start |
| Token | `Notifications.getExpoPushTokenAsync({ projectId })` on grant + app resume if registered |
| Register | `POST /v1/api/push/devices` with JWT when logged in |
| Unregister | `DELETE /v1/api/push/devices/{id}` on logout + opt-out |
| Timezone | Send IANA timezone (`America/New_York`) from `Intl.DateTimeFormat().resolvedOptions().timeZone` on register/update |
| Listeners | `addNotificationResponseReceivedListener` → deep link via existing navigation (`fucci://` / React Navigation linking) |
| Foreground | `setNotificationHandler` — show alert when app open (configurable) |
| Expo Go | Push **requires dev/production build** (not Expo Go); document in quickstart |

**app.json / native**

- iOS: push entitlement via EAS credentials (APNs key in EAS project)
- Android: FCM via `google-services.json` (EAS credentials)
- Plugin: `expo-notifications` with icon/color as needed

**Deep link payload contract**

```json
{
  "type": "debate" | "match" | "news",
  "route": "SingleDebate" | "MatchDetails" | "NewsWebView",
  "params": { "debateId": 123, "matchId": 456, "url": "https://..." }
}
```

Mobile maps `data.type` + `params` to existing stack routes.

---

### 2. API — device & preference storage

**New tables** (see `data-model.md`):

- `push_devices` — one row per (user, expo_push_token); stores platform, timezone, app version, last_seen
- `push_preferences` — per-user category toggles + master enable
- `push_delivery_log` — audit trail (campaign, user, device, status, expo ticket id)
- `push_send_ledger` — idempotency: `(user_id, campaign_key, local_date)` unique

**Endpoints** (auth required unless noted):

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/push/devices` | Register or upsert token + timezone |
| `DELETE` | `/push/devices/{id}` | Remove device (logout / opt-out) |
| `GET` | `/push/preferences` | Read toggles |
| `PUT` | `/push/preferences` | Update master + category flags |
| `POST` | `/push/devices/heartbeat` | Optional: refresh timezone + `last_seen_at` on app foreground |

**Validation**

- Reject tokens not matching Expo format (`ExponentPushToken[...]`)
- Max devices per user: 5 (drop oldest inactive)
- Guest users: no registration (push is authenticated-only v1)

---

### 3. Send pipeline (server)

**Provider:** [Expo Push API](https://docs.expo.dev/push-notifications/sending-notifications/) (`https://exp.host/--/api/v2/push/send`)

**Go package:** `services/api/internal/push/`

| Component | Responsibility |
|-----------|----------------|
| `Client` | Batch send to Expo (max 100/request), retry 429/5xx with backoff |
| `Builder` | Build `{ to, title, body, data, sound, priority }` |
| `Service` | Filter devices by prefs → dedupe ledger → send → log receipts |
| `ReceiptPoller` | Optional job: fetch receipts, mark `DeviceNotRegistered`, disable bad tokens |

**Secrets**

- `EXPO_ACCESS_TOKEN` — Expo account token for higher rate limits (required prod)

**Idempotency**

- Before send: `INSERT push_send_ledger (user_id, campaign_key, local_date) ON CONFLICT DO NOTHING`
- `campaign_key` examples: `debate:daily`, `news:daily`, `match:{fixture_id}:highlights`, `match:{fixture_id}:debates_live`
- Skip send if conflict (already sent that category today for user)

---

### 4. Scheduling (local time)

Current `internal/scheduler` runs **one daily UTC slot**. Local 6pm / 12pm requires a **timezone-aware dispatcher**.

**Recommended v1 approach: slot scanner (every 15 minutes)**

```
Cron */15 * * * * (in-process or Fly machine)
  FOR each campaign (debate@18:00, news@12:00):
    FIND users WHERE:
      push_preferences.enabled = true
      AND category_enabled = true
      AND local_time(timezone) IN [target ± 7 min window]
      AND NOT EXISTS ledger for (user, campaign, local_date)
    ENQUEUE send jobs (batch by 100 tokens)
```

**Why not one UTC cron:** Users span all offsets; 6pm local cannot be one UTC instant.

**Redis lock:** `push:scan:{campaign}:{yyyy-mm-dd}:{slot}` SetNX so multi-instance Fly apps don’t double-scan.

**Match pushes (Phase 2):** Event-driven, not slot-based — triggered from FT detector + Shorts poller (separate worker).

---

### 5. Observability & compliance

| Item | Requirement |
|------|-------------|
| Logging | Structured logs: campaign, user_id, device_id, expo_ticket, error |
| Metrics | Count sent / failed / skipped (prefs / dedupe) per campaign per day |
| Opt-out | Master toggle disables all sends; deleting device on logout |
| Apple/Google | Permission strings in native config; no silent opt-in |

---

### 6. Security

- All registration endpoints require JWT (`auth.RequireAuth`)
- Push payload must not contain PII beyond public content titles
- Rate-limit `POST /push/devices` (e.g. 10/min/user)
- Admin/test send endpoint behind `is_admin` (optional, Phase 1 nice-to-have)

---

### 7. Testing strategy

| Layer | Tests |
|-------|-------|
| API | Register upsert, pref gating, ledger dedupe, invalid token handling |
| Push client | Mock Expo HTTP; verify batching + retry |
| Mobile | Jest for payload → route mapping; manual EAS internal build for E2E |
| E2E | Expo push tool + single test device before WC |

---

### 8. Out of scope — Phase 1

- Debate/news/match **content selection** logic
- FIFA rankings table
- Article open tracking (needed for news Phase 2)
- Post-FT Shorts poller
- Web push

---

## Dependencies on existing systems

| System | Use |
|--------|-----|
| `users.id` + JWT auth | Device ownership |
| `users.locale` | Fallback if mobile timezone missing (prefer explicit `timezone` column on device) |
| Redis | Scan locks + optional send queue |
| EAS project `c02a1675-224d-4ac5-bdb6-2ed00ff33963` | Expo push credentials |
| React Navigation | Deep link targets already defined per screen |

## Open technical decisions

1. **Separate worker binary** (`services/workers`) vs extend API process — recommend **in-process scanner initially** (matches prewarm pattern), extract when match poller adds load.
2. **Receipt polling** — sync in send response vs async 15m job (recommend async job).
3. **Anonymous users** — no push v1; revisit if product wants install-level alerts.
