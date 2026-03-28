# Quickstart: Debate Tab (009)

## Prerequisites

- Repo root: `Fucci`
- Branch: `009-debate-tab-navigator`
- Backend: Go 1.22+, PostgreSQL, `services/api` running with `JWT` + DB migrations applied
- Mobile: `apps/mobile` (Expo SDK 54), `yarn install` at monorepo root

## Backend (feed endpoints)

1. Start API (from repo root or `services/api`):

   ```bash
   cd services/api && go run main.go
   ```

2. **Public browse feed** (no auth — guest / browse list):

   ```bash
   export API_ORIGIN=http://localhost:8080
   curl -sS "$API_ORIGIN/v1/api/debates/public-feed?limit=30"
   ```

3. **Authenticated feed** (JWT — per-user `new_debates` / `voted_debates`):

   Obtain a user JWT (login via `/v1/api/auth/login`), then:

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

## Debate content (news headlines)

Debates in this tab should be **generated** with **world football news / headline** context per [004-ai-debate-generator](../004-ai-debate-generator/spec.md). Ensure generation jobs or on-demand flows pull **top football headlines** into the context bundle before expecting realistic main-feed content.

## Related docs

- [spec.md](./spec.md) — user stories
- [plan.md](./plan.md) — implementation plan + sourcing note
- [contracts/debates-feed.yaml](./contracts/debates-feed.yaml) — feed contract
- [006 quickstart](../006-user-engagement-debates/quickstart.md) — comment/vote APIs
