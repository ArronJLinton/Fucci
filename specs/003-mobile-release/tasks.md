# Tasks: Automated Mobile Release (Local EAS First, Then GitHub Actions)

**Input**: Design documents from `specs/003-mobile-release/`
**Prerequisites**: plan.md, spec.md

**Organization**: Phase 1 = local EAS build and EAS Internal distribution. Phase 2 = add GitHub Actions workflow. TestFlight/Play submit deferred.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 = Local EAS Internal; US2 = GitHub Actions; Later = TestFlight/Play
- Include exact file paths in descriptions

---

## Phase 1: Local EAS Build and EAS Internal (Do This First)

**Purpose**: Document and validate building and distributing via EAS from a local machine. No GitHub Actions yet.

- [x] T001 Document required GitHub secrets (EXPO_TOKEN for CI phase) and where to create them in specs/003-mobile-release/SECRETS.md
- [x] T002 [P] Verify apps/mobile/eas.json has build.preview (distribution internal) and build.production; add or adjust for CI if needed in apps/mobile/eas.json
- [x] T003 Document trigger choice (tag pattern mobile/v*, workflow_dispatch) and distribution choice (EAS Internal first, TestFlight later) in specs/003-mobile-release/plan.md
- [ ] T004 [US1] Document local EAS build steps (run from apps/mobile: eas build --platform all --profile preview; where to find internal distribution link in Expo dashboard) in apps/mobile/README.md or specs/003-mobile-release/plan.md
- [ ] T005 [US1] Add npm script build:preview (or build:internal) in apps/mobile/package.json that runs eas build --profile preview for convenience
- [ ] T006 [US1] Run one successful local EAS build (iOS and/or Android with profile preview); verify builds in EAS dashboard and internal distribution link; optionally share link with a tester

**Checkpoint**: Maintainers can build and distribute via EAS Internal from their machine; process is documented.

---

## Phase 2: Add GitHub Actions Workflow (After Local Works)

**Purpose**: Add workflow so the same build runs on tag or manual dispatch in CI. Depends on Phase 1 being done.

- [ ] T007 Add .github/workflows/deploy-mobile.yml with triggers: push tags matching mobile/v*, workflow_dispatch
- [ ] T008 Add job in .github/workflows/deploy-mobile.yml: checkout, Set up Node (actions/setup-node), install dependencies (yarn install), install EAS CLI (npm install -g eas-cli or expo/eas-action)
- [ ] T009 Wire EXPO_TOKEN or EAS_TOKEN as environment variable for EAS CLI in .github/workflows/deploy-mobile.yml (secret reference only; no values in file)
- [ ] T010 [US2] Add EAS build step in .github/workflows/deploy-mobile.yml: eas build --platform all --profile preview --non-interactive (or --wait); set working-directory to apps/mobile or repo root per EAS project config
- [ ] T011 [US2] Run one successful workflow (manual dispatch or tag mobile/v0.0.1-test); verify builds in EAS dashboard and internal distribution link

**Checkpoint**: Tag or manual run in GitHub produces EAS Internal builds; only EXPO_TOKEN required.

---

## Phase 3: Polish & Cross-Cutting Concerns

**Purpose**: Release process documentation

- [ ] T012 [P] Add release checklist (local: run eas build from apps/mobile; CI: bump version, tag mobile/v*, push, wait for EAS build and grab internal link) to specs/003-mobile-release/plan.md or apps/mobile/README.md
- [ ] T013 [P] Optional: Add workflow step to post status to Slack/Discord or GitHub commit status in .github/workflows/deploy-mobile.yml
- [ ] T014 Update README.md or apps/mobile/README.md with pointer to specs/003-mobile-release/ for mobile release (local EAS + optional GitHub Actions) and required secrets

---

## Later Phase (Deferred): TestFlight for iOS

**Goal**: Add submit to TestFlight; requires Apple credentials.

- [ ] T015 [Later] Add EAS submit step for iOS (eas submit --platform ios --latest --non-interactive) after build in .github/workflows/deploy-mobile.yml
- [ ] T016 [Later] Wire Apple credentials as GitHub secrets; document in SECRETS.md
- [ ] T017 [Later] Run workflow and verify iOS build is submitted to TestFlight

---

## Dependencies & Execution Order

- **Phase 1**: T001–T003 done; T004–T006 = local EAS (docs, script, one local build). No GitHub Actions yet.
- **Phase 2**: Depends on Phase 1. Add workflow file and run in CI.
- **Phase 3**: Depends on Phase 2 (or Phase 1 if skipping CI for a while).
- **Later**: TestFlight (T015–T017) when scheduled.

---

## Implementation Strategy

### Order of work

1. **Phase 1** (local EAS): Document local build steps, add build:preview script, run one local build and verify EAS Internal link.
2. **Phase 2** (GitHub Actions): Add deploy-mobile.yml, EXPO_TOKEN secret, run workflow and verify.
3. **Phase 3**: Release checklist and README pointer.
4. **Later**: TestFlight when ready.

### Summary

| Phase              | Task count | Description                          |
|--------------------|-----------:|--------------------------------------|
| Phase 1 Local EAS  | 6          | Docs, script, one local build (T001–T003 done; T004–T006) |
| Phase 2 GitHub Actions | 5      | Workflow file, build step, verify    |
| Phase 3 Polish     | 3          | Release checklist, README            |
| Later (TestFlight) | 3          | Deferred                             |

**Suggested order**: Complete Phase 1 (T004–T006), then Phase 2 (T007–T011), then Phase 3. TestFlight in a later iteration.
