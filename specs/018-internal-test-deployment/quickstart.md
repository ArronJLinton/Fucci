# Quickstart: Internal Test Deployment (Android + iOS)

This runbook is the execution-ready task list for shipping Fucci to internal testing on both platforms.

## Deployment Execution Notes

- Primary working directory for mobile release commands: `apps/mobile/`.
- **iOS bundle ID (canonical):** `com.magistridev.fucci` — set only in `apps/mobile/app.json` → `expo.ios.bundleIdentifier` (EAS `eas.json` build profiles do not support `ios.bundleIdentifier`).
- Run all release commands using organization-owned Expo and store accounts.
- Keep build profile names fixed: `development`, `preview`, `production`.
- Keep OTA channels fixed: `dev`, `staging`, `production`.
- Always execute `yarn preflight:release` before any EAS build or submit command.

## EAS profile ↔ OTA channel mapping (US2)

| EAS build profile | OTA channel (`eas update`) | Typical use |
|---|---|---|
| `development` | `dev` | Dev client / local iteration |
| `preview` | `staging` | Internal QA (Play internal + TestFlight internal) |
| `production` | `production` | Store-ready binaries |

Submit defaults for internal test tracks are configured under `submit.preview` in `apps/mobile/eas.json` (Android internal track, iOS TestFlight via ASC identifiers). Use `--profile preview` with `eas submit` for gated internal releases.

## Dependency Legend

- **[BLOCKER]** must be completed before dependent tasks.
- **Depends on:** task IDs that must be done first.

## CI Secret Inventory (GitHub + Expo)

| Secret | Scope | Required For | Source of Truth |
|---|---|---|---|
| `EXPO_TOKEN` | GitHub Actions | `eas build`, `eas submit`, `eas update` | Expo account token (org-owned) |
| `GOOGLE_SERVICE_ACCOUNT_JSON` | GitHub Actions | Android submit to Play internal track | Google Cloud service account |
| `ASC_API_KEY_ID` | GitHub Actions | iOS submit/TestFlight automation | App Store Connect API key |
| `ASC_API_ISSUER_ID` | GitHub Actions | iOS submit/TestFlight automation | App Store Connect API key |
| `ASC_API_KEY_P8` | GitHub Actions | iOS submit/TestFlight automation | App Store Connect API key |
| `FCM_SERVER_KEY` / delegated push credential | Backend/Secrets Manager | Android push delivery path | Firebase project secrets |
| `APNS_KEY_P8` | Expo/EAS Secrets | iOS push delivery path | Apple Developer APNs key |
| `SUPABASE_URL` | GitHub/Runtime | Runtime backend config | Supabase project settings |
| `SUPABASE_ANON_KEY` | GitHub/Runtime | Runtime backend config | Supabase project settings |

## Environment Variable Reference (Mobile Deploy Jobs)

| Variable | Environment | Example Use |
|---|---|---|
| `APP_ENV=development` | Development builds | Local/dev-client testing |
| `APP_ENV=staging` | Preview builds | Internal QA/test tracks |
| `APP_ENV=production` | Production builds | Store-ready builds |
| `EAS_BUILD_PROFILE` | CI build jobs | Select `development`/`preview`/`production` |
| `EAS_UPDATE_CHANNEL` | OTA publish jobs | Select `dev`/`staging`/`production` |

## Command Baseline (Normalized)

From `apps/mobile/`:

```bash
# Preflight
yarn preflight:release

# Local environment switching
yarn env:dev
yarn env:staging
yarn env:prod

# Build (internal and production)
npx eas-cli build --platform android --profile preview --non-interactive
npx eas-cli build --platform ios --profile preview --non-interactive
npx eas-cli build --platform android --profile production --non-interactive
npx eas-cli build --platform ios --profile production --non-interactive

# Submit (manual-gated in CI, but executable locally if needed)
npx eas-cli submit --platform android --profile preview --non-interactive
npx eas-cli submit --platform ios --profile preview --non-interactive
```

## Phase A — Organization, Ownership, and Access

- [ ] **A1 [BLOCKER]** Confirm organization ownership for Google Play, Apple Developer, Firebase, Expo, and GitHub.
  - Depends on: none
  - Dependency note: All credential/app-record tasks require correct owner permissions.
- [ ] **A2 [BLOCKER]** Configure role-based access (least privilege) for Mobile, DevOps, QA.
  - Depends on: A1
  - Dependency note: CI secret provisioning and submission permissions require this.

## Phase B — Android Setup (Google + Firebase + Play)

- [ ] **B1 [BLOCKER]** Create/verify Google Play Console organization and payment/compliance setup.
  - Depends on: A1
  - Dependency note: Cannot create app record or internal track without this.
- [ ] **B2 [BLOCKER]** Create Android app in Play Console (`applicationId` frozen).
  - Depends on: B1
  - Dependency note: `applicationId` must match Expo/EAS and Firebase Android app.
- [ ] **B3 [BLOCKER]** Create Firebase project (or attach existing) and register Android app with same package name.
  - Depends on: A1, B2
  - Dependency note: FCM credentials and `google-services.json` depend on this.
- [ ] **B4 [BLOCKER]** Enable Firebase Cloud Messaging and generate service account key (if required by workflow).
  - Depends on: B3
  - Dependency note: Expo push for Android and backend notification sending depend on this.
- [ ] **B5** Store Android secrets in secret manager/GitHub/Expo (never commit keys).
  - Depends on: B4
  - Dependency note: CI build and push validation blocked until secrets are available.
- [ ] **B6 [BLOCKER]** Configure Play Internal Testing track + tester groups.
  - Depends on: B2
  - Dependency note: No tester distribution without internal track setup.

## Phase C — iOS Setup (Apple Developer + APNs + App Store Connect)

- [ ] **C1 [BLOCKER]** Enroll in Apple Developer Program (organization account preferred).
  - Depends on: A1
  - Dependency note: Required for signing, bundle IDs, provisioning, and TestFlight.
- [ ] **C2 [BLOCKER]** Create App ID (Bundle Identifier) in Apple Developer portal.
  - Depends on: C1
  - Dependency note: Must match Expo iOS bundle identifier and App Store Connect app record.
- [ ] **C3 [BLOCKER]** Create APNs Auth Key (`.p8`), capture Key ID + Team ID + Bundle ID mapping.
  - Depends on: C1, C2
  - Dependency note: iOS push notifications blocked without APNs credentials.
- [ ] **C4 [BLOCKER]** Create App Store Connect app record for Fucci (same bundle ID).
  - Depends on: C2
  - Dependency note: iOS submissions/TestFlight blocked until app record exists.
- [ ] **C5** Configure TestFlight internal testing groups and add testers.
  - Depends on: C4
  - Dependency note: Tester distribution blocked until group setup is complete.

## Phase D — Expo / EAS Project Configuration

- [ ] **D1 [BLOCKER]** Link Expo project to EAS (`eas init`) and confirm `projectId`.
  - Depends on: A1
  - Dependency note: All EAS build/submit/update tasks require project linkage.
- [ ] **D2 [BLOCKER]** Verify `app.json` identifiers:
  - Android `package` matches Play/Firebase.
  - iOS `bundleIdentifier` matches Apple/App Store Connect.
  - Depends on: B2, B3, C2, D1
  - Dependency note: Mismatch causes build/signing/submission failures.
- [ ] **D3 [BLOCKER]** Create/update `apps/mobile/eas.json` build profiles:
  - `development`: dev client, internal distribution.
  - `preview`: internal distribution for QA/testing.
  - `production`: store-ready profile.
  - OTA channel mapping: `development -> dev`, `preview -> staging`, `production -> production`.
  - Depends on: D1, D2
  - Dependency note: CI and OTA channel mapping depend on stable profiles.
- [ ] **D3.1** Validate local preflight commands in `apps/mobile/package.json`:
  - `yarn preflight:typecheck`
  - `yarn preflight:doctor`
  - `yarn preflight:eas`
  - `yarn preflight:release`
- [ ] **D4** Define `submit` config for Android internal track and iOS TestFlight.
  - Depends on: B6, C4, D3
  - Dependency note: Automated submission blocked without valid submit config.

## Phase E — Credential Management and Backup

- [ ] **E1 [BLOCKER]** Generate/import Android keystore via EAS credentials.
  - Depends on: D3
  - Dependency note: Android app updates depend on same keystore forever.
- [ ] **E2 [BLOCKER]** Back up Android keystore + alias/passwords in team secret manager.
  - Depends on: E1
  - Dependency note: Catastrophic release blocker if lost.
- [ ] **E3 [BLOCKER]** Configure iOS signing assets (Distribution cert + provisioning) through EAS credentials.
  - Depends on: C1, C2, C4, D3
  - Dependency note: iOS archive/submission blocked without signing assets.
- [ ] **E4 [BLOCKER]** Upload APNs key to Expo/EAS notifications config and verify mapping.
  - Depends on: C3, D1
  - Dependency note: iOS push testing blocked without APNs key.
- [ ] **E5** Create credential audit record (owner, creation date, rotation policy, backup location).
  - Depends on: E2, E3, E4
  - Dependency note: Compliance and incident recovery depend on this.

## Phase F — OTA via EAS Update

- [ ] **F1 [BLOCKER]** Define runtimeVersion policy (appVersion or nativeVersion) and document it.
  - Depends on: D3
  - Dependency note: Incorrect runtimeVersion can break OTA compatibility.
  - Policy baseline: use app version as runtime boundary and update binary whenever native modules change.
- [ ] **F2 [BLOCKER]** Map update channels:
  - `development` -> `dev`
  - `preview` -> `staging`
  - `production` -> `production`
  - Depends on: D3, F1
  - Dependency note: Channel leakage can expose wrong update to testers/users.
- [ ] **F3** Verify OTA publish flow for staging channel (`eas update --channel staging`) on internal build.
  - Depends on: F2, G3, G4
  - Dependency note: Requires installed internal test binary built with matching runtime.

## Phase G — GitHub Actions CI/CD

- [ ] **G1 [BLOCKER]** Create Expo automation token and add GitHub secret `EXPO_TOKEN`.
  - Depends on: D1, A2
  - Dependency note: CI cannot call EAS without token.
- [ ] **G2 [BLOCKER]** Add required CI secrets (store/API keys, service accounts, env vars).
  - Depends on: B5, C3, E2, E3, A2
  - Dependency note: Build and submit jobs fail without secret set.
- [ ] **G3 [BLOCKER]** Implement workflow job: Android preview build (`eas build --platform android --profile preview --non-interactive`).
  - Depends on: D3, G1, G2
  - Dependency note: Needed before Android submit and OTA validation.
- [ ] **G4 [BLOCKER]** Implement workflow job: iOS preview build (`eas build --platform ios --profile preview --non-interactive`).
  - Depends on: D3, G1, G2
  - Dependency note: Needed before iOS submit and OTA validation.
- [ ] **G5 [BLOCKER]** Implement gated submit jobs:
  - Android internal track (`eas submit --platform android --profile preview`).
  - iOS TestFlight (`eas submit --platform ios --profile preview`).
  - Depends on: D4, G3, G4
  - Dependency note: Requires completed build artifacts and store app records.
- [ ] **G6** Add workflow protections:
  - Trigger strategy (`workflow_dispatch` + optional branch gates).
  - Environment approvals for submit.
  - Artifact/log retention.
  - Depends on: G5
  - Dependency note: Reduces accidental submissions and improves auditability.

## Phase H — End-to-End Validation (Required Before “Ready”)

- [ ] **H1 [BLOCKER]** Install latest internal Android build from Play Internal Testing; verify install/launch/login/feed/match/debate flows.
  - Depends on: G5
- [ ] **H2 [BLOCKER]** Install latest iOS build from TestFlight; verify same critical flows.
  - Depends on: G5
- [ ] **H3 [BLOCKER]** Validate push notifications on both platforms:
  - kick-off reminder push
  - debate reply push
  - foreground/background tap behavior
  - Depends on: B4, E4, H1, H2
  - Dependency note: Push is in core scope; cannot mark internal deployment complete without pass.
- [ ] **H4** Validate OTA staging update delivery and rollback procedure.
  - Depends on: F3, H1, H2
- [ ] **H5 [BLOCKER]** Final readiness sign-off (Mobile + QA + Platform).
  - Depends on: H1, H2, H3, H4

## Suggested GitHub Actions Job Graph

Implemented in `.github/workflows/mobile-internal-deploy.yml`:

- `validate-secrets` → `build-android-preview` and `build-ios-preview` (parallel)
- On `workflow_dispatch` with **Run gated submit** enabled → `submit-android-preview` and `submit-ios-preview` (each requires Environment approval: `mobile-internal-submit`)

Push to `main` (paths under `apps/mobile/` or this workflow) runs validate + both preview builds only; submit does not run on push.

## Definition of Done

- Both internal tracks receive installable builds from CI.
- Push notifications pass end-to-end on Android and iOS.
- OTA staging channel verified on installed internal builds.
- Signing credentials are backed up and documented.
- Release runbook and dependency notes are up to date.

## US1 Platform and Push Setup Runbook

### Google Play Console Setup (T013)

1. Create or verify organization-owned Play Console account.
2. Create app record with package name `com.magistridev.fucci`.
3. Complete policy, app access, and content rating placeholders required for internal testing.
4. Configure internal testing track and create tester group.
5. Capture blocker dependencies:
   - blocks Android submit until internal track exists
   - package name must match `apps/mobile/app.json`

### Firebase + FCM Setup (T014)

1. Create/select organization-owned Firebase project for Fucci mobile.
2. Register Android app using package `com.magistridev.fucci`.
3. Enable Cloud Messaging and create server/delegated credential used by backend push path.
4. Store credential material in approved secret store (not source control).
5. Capture blocker dependencies:
   - blocks Android push validation until credential exists
   - blocks CI submit if service account JSON is missing

### Apple Developer Enrollment + App ID (T015)

1. Confirm Apple Developer Program enrollment under organization account.
2. Create App ID for `com.magistridev.fucci`.
3. Ensure Team role assignments include backup release operator.
4. Capture blocker dependencies:
   - blocks provisioning and signing asset creation
   - blocks App Store Connect app creation when bundle ID mismatches

### APNs Key Creation and Storage (T016)

1. Create APNs Auth Key (`.p8`) in Apple Developer account.
2. Record Key ID and Team ID alongside bundle identifier.
3. Upload APNs key to Expo/EAS credential store.
4. Back up `.p8` and metadata in vault with dual-admin access.
5. Capture blocker dependencies:
   - blocks iOS push testing and debate/kick-off notification validation

### App Store Connect + TestFlight Internal Groups (T017)

1. Create App Store Connect app with bundle ID `com.magistridev.fucci`.
2. Configure internal TestFlight groups and assign tester emails.
3. Verify at least one internal tester can access the assigned group.
4. Capture blocker dependencies:
   - blocks iOS internal distribution and submit verification

### US1 Verification Checklist (T019)

- [ ] Play Console app record exists and package matches `com.magistridev.fucci`.
- [ ] Internal testing track configured with active tester group.
- [ ] Firebase Android app exists and FCM credential is stored securely.
- [ ] Apple App ID exists for `com.magistridev.fucci`.
- [ ] APNs `.p8` key uploaded to Expo/EAS and backed up in vault.
- [ ] App Store Connect app record exists with internal TestFlight group.
- [ ] Blocker dependencies documented and linked to setup owner.

## US2 — EAS credential lifecycle (T022)

Execute in order; each step blocks the next until credentials are valid in Expo.

1. **Android keystore**: `eas credentials` → Android → set up or import upload keystore; record keystore backup per Phase E.
2. **iOS distribution**: `eas credentials` → iOS → distribution certificate + provisioning profile for App Store (or let EAS manage).
3. **APNs for push**: upload APNs key to EAS (notifications) and confirm bundle ID mapping matches `apps/mobile/app.json`.
4. **Verification**: run `yarn preflight:eas` and confirm project resolves credentials without interactive prompts in CI.

## US2 — Release channel and runtime compatibility (T023)

Before publishing an OTA update:

- Confirm the target binary was built with the **same** `runtimeVersion` policy as the update (see Phase F).
- Confirm `EAS_UPDATE_CHANNEL` matches the build profile (`preview` builds consume `staging` only).
- After native dependency or Expo SDK upgrades, ship a **new store binary** before relying on OTA for that branch.
- Smoke-test: install preview build, then confirm `eas update --channel staging` applies only to that runtime.

## US2 — OTA publish and rollback — `staging` channel (T024)

From `apps/mobile/` (after a successful `preview` build with matching runtime):

```bash
# Publish JS/asset update to internal testers (staging channel)
npx eas-cli update --channel staging --message "describe change" --non-interactive

# List recent updates (inspect IDs/branches)
npx eas-cli update:list --channel staging --non-interactive
```

Rollback (republish previous known-good bundle or use Expo dashboard to roll back the channel; document the rollback ticket ID in release notes).

## US2 — Validation checklist (T026)

- [ ] `apps/mobile/eas.json` defines `submit.preview` with Android `track: internal` and iOS ASC fields (placeholders replaced for real submits).
- [ ] Build profiles `development` / `preview` / `production` each set `channel` to `dev` / `staging` / `production`.
- [ ] `eas build --profile preview` succeeds for Android and iOS without credential prompts in CI context.
- [ ] `eas submit --profile preview` targets internal track / TestFlight per org policy (manual gate in CI).
- [ ] `eas update --channel staging` applies to preview-built binaries only; runtime mismatch test performed once per native change.

## US3 — GitHub Actions workflow (T027–T033)

Workflow file: `.github/workflows/mobile-internal-deploy.yml`

### Required repository secrets and setup (T031)

| Secret | Where | Purpose |
|--------|--------|---------|
| `EXPO_TOKEN` | Repository or Environment secrets | `eas build`, `eas submit`, `eas update` from CI |

Optional (store automation beyond EAS-managed credentials): mirror entries from **CI Secret Inventory** (`GOOGLE_SERVICE_ACCOUNT_JSON`, App Store Connect API key fields) if you wire them into `eas submit` or separate steps.

**One-time setup**

1. In Expo: create an organization access token suitable for CI; add as `EXPO_TOKEN` in GitHub (**Settings → Secrets and variables → Actions**).
2. Create GitHub Environment **`mobile-internal-submit`** with **required reviewers** so submit jobs cannot run without approval.
3. Run workflow manually: **Actions → Mobile EAS (preview) → Run workflow**; enable **Run gated submit** only after both preview builds succeed and you intend to upload to Play internal / TestFlight.

### CI job graph and dependencies (T032)

```text
validate-secrets
├── build-android-preview   (needs: validate-secrets)
├── build-ios-preview       (needs: validate-secrets)

# workflow_dispatch only, input run_submit == true, Environment: mobile-internal-submit
submit-android-preview      (needs: build-android-preview)
submit-ios-preview          (needs: build-ios-preview)
```

- **Fail-fast**: `validate-secrets` exits with an error if `EXPO_TOKEN` is empty.
- **Artifacts**: each build job uploads a small `ci-artifacts/metadata.json` with **retention-days: 30** and writes a short summary to the job log / `GITHUB_STEP_SUMMARY`.

### US3 run validation checklist (T033)

- [ ] `validate-secrets` passes on a branch where `EXPO_TOKEN` is configured.
- [ ] `build-android-preview` completes and produces an EAS Android preview build.
- [ ] `build-ios-preview` completes and produces an EAS iOS preview build.
- [ ] With **Run gated submit** checked, Environment protection prompts reviewers before `submit-*` jobs run.
- [ ] After approval, `eas submit --latest --profile preview` succeeds for Android and iOS (store accounts and EAS credentials already configured).

## Credential Backup & Custody Procedure

1. Store Android keystore, alias, and passwords in team-managed vault (dual-admin access).
2. Store APNs key (`.p8`), Key ID, Team ID in team-managed vault and Expo secrets.
3. Record custody metadata:
   - creator
   - backup owner
   - created date
   - last restore drill date
4. Run a restore drill at least once per quarter and log outcome in this runbook.
5. Never keep release credentials only on a personal machine.
