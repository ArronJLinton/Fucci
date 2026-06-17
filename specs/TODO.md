# Fucci — Project TODO

Project-wide backlog. Anything that doesn't yet have a dedicated `specs/NNN-*/`
folder lives here. When an item grows into real work, promote it into its own
Speckit spec folder and remove it from this list.

## How to use this file

- Add a bullet under the most appropriate section (or add a new section if
  none fits — keep titles short).
- Each item should be self-contained: someone picking it up six months from
  now should be able to start without re-reading chat history.
- Priority hint in brackets: `[P0]` blocker / urgent · `[P1]` important ·
  `[P2]` nice-to-have · `[P3]` future.
- When something ships, delete it (git history is the audit log).

Sections (add new ones as needed):

- Features
- Bug fixes
- Performance
- Accessibility
- Infrastructure / DevOps
- Cron jobs & background workers
- Observability & monitoring
- Security & privacy
- Tech debt & refactors
- Testing
- Documentation
- Dependencies
- Branch hygiene

---

## Features

- **[P1] Post-match debate generation.** Pre-match is shipped (see commit
  `7700bfea4`); post-match is currently a manual on-demand call. Add a poller
  that wakes ~N minutes after a fixture reaches `FT`/`AET`/`PEN` and POSTs to
  `/v1/api/debates/generate-set` with `debate_type=post_match`. Reuse the
  `PrewarmJob` patterns (SetNX dedup, loopback HTTP) where possible.

## Bug fixes

- *(none currently tracked)*

## Performance

- *(none currently tracked)*

## Accessibility

- *(none currently tracked)*

## Infrastructure / DevOps

- **[P2] API-Football subscription.** Free tier only exposes seasons
  2022–2024, which is why World Cup 2026 fixture queries returned empty during
  testing on the `world-cup-only-mode` branch. Options:
  1. Upgrade to a paid plan that includes 2026.
  2. Switch to a staging API / mock during pre-launch.
  Decision needed before the World Cup goes live.
- **[P2] Rename remote slug — done locally.** Origin was migrated from
  `ArronJLinton/fucci.git` → `ArronJLinton/Fucci.git`. Anyone else with a
  local clone should run:

  ```bash
  git remote set-url origin git@github.com:ArronJLinton/Fucci.git
  ```

  Add a note to the README / onboarding doc when convenient.

## Cron jobs & background workers

- **[P2] Graduate scheduler from in-process to `services/workers/`.** The
  current pre-match pre-warm runs as an in-process goroutine inside
  `services/api` (Option B from the architectural decision on 2026-06-16).
  When we add a second background job (post-match generation, leaderboards,
  cleanup tasks, etc.), promote it to a dedicated workers binary so the API
  process stays single-purpose.
  Ref: `services/api/internal/scheduler/`, `services/workers/main.go` (TODO
  scaffold), commit `7700bfea4`.
- **[P2] Extend `PREWARM_LEAGUE_IDS` for August.** When club seasons start,
  change the env var from `"1"` (FIFA World Cup only) to
  `"1,39,140,135,78,61,2"` (EPL / La Liga / Serie A / Bundesliga / Ligue 1 /
  UCL + WC) so the pre-warm covers all leagues we render. No code change
  required — just an env update on Fly.

## Observability & monitoring

- **[P2] Structured logging for scheduler & pre-warm.** Currently uses
  `log.Printf`; the rest of the API uses `otelzap`. Convert
  `services/api/internal/scheduler/scheduler.go` and
  `services/api/internal/api/prewarm.go` so log lines are structured and ship
  to whatever sink we wire up next.
- **[P2] Pre-warm metrics.** Surface `fixtures_seen`, `news_warmed`,
  `debates_ok`, and `errors` per run as counters/gauges so silent
  failures become visible (e.g. upstream returning 0 fixtures for a league
  on multiple days in a row).

## Security & privacy

- *(none currently tracked)*

## Tech debt & refactors

- **[P1] Flip `WORLD_CUP_ONLY_MODE` back off in August.** Set
  `WORLD_CUP_ONLY_MODE = false` in `apps/mobile/src/config/featureFlags.ts`.
  Once the next full season is underway, consider removing the flag and the
  surrounding `if (WORLD_CUP_ONLY_MODE)` branches entirely (search for
  `WORLD_CUP_ONLY_MODE` references across `apps/mobile/src/`).
- **[P3] Simplify `getMatchNews` stale-cache fallback.** The 503-with-cached-
  payload branch is documented as unreachable in
  `services/api/internal/api/news_test.go`; either delete the dead code or
  re-add a deliberate stale-on-failure path.

## Testing

- **[P2] Direct unit test for `Config.FetchMatchesCached`.** Currently
  exercised indirectly via `TestGetMatchesSeasonResolutionAndCache` and
  `TestGetMatchesQueryValidation` (handler tests). Add a focused test that
  passes `nil` vs non-`nil` league/season pointers and asserts cache key
  shape + upstream URL formation.
- **[P2] Direct test for `DebateDataAggregator.fetchNewsHeadlines`.** The
  new shared-cache signature is only validated transitively by the prewarm
  e2e tests. A direct test in `debate_data_aggregator_test.go` would catch
  signature/contract regressions earlier.
- **[P3] `Scheduler.Start`/`Stop` lifecycle test.** Skipped intentionally
  while there's only one job; add when we ship a second.

## Documentation

- *(none currently tracked)*

## Dependencies

- *(none currently tracked)*

## Branch hygiene

- **[P1] Rebase `world-cup-only-mode` on latest `main`.** As of 2026-06-17,
  `origin/main` advanced from `600926a31` → `cd00085e6` while we were
  building on this branch. Rebase before opening a PR (or before next push)
  to keep the diff focused on the WC-mode work.
- **[P2] Review `origin/runtime-fixes` (`73d86f86c..468334c9f`).** Active
  branch that picked up new commits — confirm it's not blocked on anything
  we touched.
- **[P2] Triage `cursor/critical-bug-investigation-*` branches.** Three new
  branches landed on origin (`-6420`, `-8bbf`, `-a629`). Look at each, decide
  whether to merge, close, or extract the diagnostic notes.
