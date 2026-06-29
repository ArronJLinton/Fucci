# Implementation Plan: Push Notifications — Phase 1 Infrastructure

**Branch**: `feat/push-notifications`  
**Spec**: `specs/010-push-notifications/spec.md`

## Phase 1 tasks (infra only)

### A. Credentials & mobile native

1. Configure APNs + FCM in EAS (see `specs/018-internal-test-deployment` push notes)
2. Add `expo-notifications` + plugin to `apps/mobile`
3. Add permission UX in Settings (master + category toggles UI shell)
4. Implement `PushRegistrationService`: token fetch → API register on login; delete on logout

### B. Database & API

1. Goose migrations: `push_devices`, `push_preferences`, `push_send_ledger`, `push_delivery_log`
2. sqlc queries + handlers under `/v1/api/push/*`
3. Wire routes in `api.go` with `auth.RequireAuth`

### C. Send pipeline

1. `internal/push/client.go` — Expo HTTP client
2. `internal/push/service.go` — prefs filter, ledger insert, batch send, log
3. `EXPO_ACCESS_TOKEN` in config + `.env.example`
4. Admin-only `POST /push/test` (single device) for QA

### D. Scheduler foundation

1. `PushSlotScanner` job — runs every 15m, **no-op body** in Phase 1 (logs only) OR sends test campaign in staging
2. Redis lock per scan window
3. Helper: `LocalTimeInTimezone(tz string) (time.Time, error)` + unit tests for DST edges

### E. Deep linking

1. `apps/mobile/src/navigation/pushLinking.ts` — map notification `data` → navigate
2. Wire in `NavigationContainer` linking config + notification response listener

### F. Tests

1. API handler tests (register, prefs, dedupe)
2. Push client mock tests
3. Mobile unit tests for route mapping

## Phase 2 preview (not in this branch scope)

- `DebateDailyCampaign` selector (engagement + unvoted + boost)
- `NewsDailyCampaign` selector (heuristic + opened filter)
- `MatchFTCampaign` (Shorts poller + 2/day cap + FIFA ranks table)
- Article open tracking endpoint

## Definition of done — Phase 1

- [ ] Physical device on EAS internal build receives test push from admin endpoint
- [ ] Token persists across app restart; removed on logout
- [ ] Preferences gate sends (disabled category → ledger `skipped_prefs`)
- [ ] Tapping notification opens correct screen
- [ ] Dedupe ledger prevents duplicate test sends same local day
