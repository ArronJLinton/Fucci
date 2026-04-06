# Quickstart: Internal Test Deployment (Android + iOS)

This runbook is the execution-ready task list for shipping Fucci to internal testing on both platforms.

## Deployment Execution Notes

- Primary working directory for mobile release commands: `apps/mobile/`.
- Run all release commands using organization-owned Expo and store accounts.
- Keep build profile names fixed: `development`, `preview`, `production`.
- Keep OTA channels fixed: `dev`, `staging`, `production`.
- Always execute `yarn preflight:release` before any EAS build or submit command.

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

- `lint_test` -> `android_preview_build` and `ios_preview_build`
- `android_preview_build` + `ios_preview_build` -> `submit_preview` (manual approval)
- `submit_preview` -> `notify_internal_testers`

## Definition of Done

- Both internal tracks receive installable builds from CI.
- Push notifications pass end-to-end on Android and iOS.
- OTA staging channel verified on installed internal builds.
- Signing credentials are backed up and documented.
- Release runbook and dependency notes are up to date.

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
