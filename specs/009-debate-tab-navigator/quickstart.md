# Quickstart: Debate Tab (009)

## Prerequisites

- Repo root: `Fucci`
- Branch: `009-debate-tab-navigator`
- Backend: Go 1.22+, PostgreSQL, `services/api` running with `JWT` + DB migrations applied
- Mobile: `apps/mobile` (Expo SDK 54), `yarn install` at monorepo root

## Backend (feed endpoint — after implementation)

1. Start API (from repo root or `services/api`):

   ```bash
   cd services/api && go run main.go
   ```

2. Obtain a user JWT (login via `/v1/api/auth/login`).

3. Call feed (expected once implemented):

   ```bash
   curl -sS -H "Authorization: Bearer $TOKEN" \
     "$API_ORIGIN/v1/api/debates/feed?new_limit=10&voted_limit=10"
   ```

## Mobile

1. Set `API_BASE_URL` in `apps/mobile` env (see `app.json` / `expo-constants`).

2. Start Expo:

   ```bash
   cd apps/mobile && yarn dev
   ```

3. After feature implementation: open app → tap **Debates** tab → verify **New** section above **My Activity**, swipe on hero card, tap voted row → detail.

## Related docs

- [spec.md](./spec.md) — user stories
- [contracts/debates-feed.yaml](./contracts/debates-feed.yaml) — feed contract
- [006 quickstart](../006-user-engagement-debates/quickstart.md) — comment/vote APIs
