# Implementation Plan: Automated Mobile Release (EAS Internal First)

**Branch**: `003-mobile-release` (or current release branch)  
**Spec**: [spec.md](./spec.md)  
**Status**: For review before coding

---

## 1. Summary

**Deploy via EAS locally first**, then add GitHub Actions. First iteration: (1) Document and validate **local** EAS build and EAS Internal distribution (run `eas build --profile preview` from the repo, share internal link with testers). (2) Once that works, add a GitHub Actions workflow that runs the same build on tag or manual dispatch. No App Store or Play Store submit in this iteration — testers install from Expo’s internal distribution link. **TestFlight (iOS) and Play Store internal (Android) submit are deferred to a later phase.**

**Out of scope for this plan**: Production App Store / Play Store submission; TestFlight/Play submit (later phase).

---

## 2. Current State

- **API**: `.github/workflows/deploy-api.yml` deploys the Go API to Fly.io on push to `main`/`master` when `services/api/**` changes.
- **Mobile**: 
  - `apps/mobile/` is an Expo (SDK 54) app with existing `eas.json`.
  - `eas.json` defines `build.development`, `build.preview` (distribution: internal), and `build.production`; `submit.production` exists for a future TestFlight/Play phase.
  - `package.json` has `build:ios` and `build:android` scripts (production profile).
- **Gap**: No automated job that runs EAS Build on a tag or button click; no internal distribution pipeline.

---

## 3. Proposed Solution (First Iteration: EAS Internal Only)

### 3.1 Trigger options

| Option | Trigger | Use case |
|--------|--------|----------|
| **A** | Push tag `mobile/v*` (e.g. `mobile/v1.0.2`) | Versioned internal releases |
| **C** | `workflow_dispatch` only (manual “Run workflow”) | One-off builds |

**Recommendation**: **A + C** — Tag `mobile/v*` plus `workflow_dispatch`. No trigger on push to `main`.

### 3.2 Workflow shape (first iteration)

1. **Trigger**: On push of tag `mobile/v*` and/or `workflow_dispatch`.
2. **Job steps**:
   - Checkout repo.
   - Set up Node; install dependencies (e.g. `yarn install` from repo root or `apps/mobile`).
   - Install EAS CLI.
   - **Build**: `eas build --platform all` (or `ios` and `android` separately) with profile **preview** (EAS Internal distribution). Use `--non-interactive`; optionally `--wait` to wait for build completion in the same job.
3. **Secrets**: Only **EXPO_TOKEN** (or EAS_TOKEN). No Apple or Google credentials in this iteration.

Builds appear in the EAS dashboard and are available via the internal distribution link; no submit step.

### 3.3 EAS Internal vs TestFlight (later)

- **This iteration**: Use `build.preview` (or a dedicated internal profile) with `distribution: "internal"`. Build only; no `eas submit`. Testers use Expo’s link.
- **Later (TestFlight)**: Add `eas submit` for iOS to TestFlight; will require Apple credentials (App Store Connect API key or Apple ID + app-specific password). See [SECRETS.md](./SECRETS.md) “Later: TestFlight / Play” when that phase is scheduled.

### 3.4 Where the workflow file lives

- **File**: `.github/workflows/deploy-mobile.yml`.
- **Naming**: “deploy” here means “build and publish to EAS Internal”; consistent with `deploy-api.yml`.

### 3.5 Configuration decisions (Phase 1)

- **Trigger**: **Option A + C** — Tag `mobile/v*` and `workflow_dispatch`. No push to `main`.
- **Distribution (first iteration)**: **EAS Internal** via build profile `preview` (or equivalent). No TestFlight/Play submit; Apple/Google credentials deferred.

---

## 4. Prerequisites (before coding)

- [ ] **Expo account**: EAS project linked to this repo (`eas build` runs successfully from a maintainer’s machine).
- [ ] **GitHub**: Secret `EXPO_TOKEN` (or `EAS_TOKEN`) created. No Apple or Google secrets required for this iteration.
- [ ] **Tag strategy**: Tags `mobile/v*` (e.g. `mobile/v1.0.0`). Document in release checklist.

---

## 5. Implementation Phases

### Phase 1: Local EAS build and EAS Internal distribution (do this first)

1. Ensure `eas.json` has a `preview` (or internal) profile with `distribution: "internal"` for both iOS and Android.
2. Document **local** EAS build steps in `apps/mobile/README.md` or `specs/003-mobile-release/` (e.g. run from `apps/mobile`: `eas build --platform all --profile preview`; where to find the internal distribution link in Expo dashboard).
3. Add npm scripts in `apps/mobile/package.json` if helpful (e.g. `build:preview` or `build:internal` that runs `eas build --profile preview`).
4. Run at least one successful local build (iOS and/or Android); confirm builds appear in EAS and are available via internal distribution link; share link with a tester if possible.

**Deliverable**: Any maintainer can build and distribute via EAS Internal from their machine; process is documented.

### Phase 2: Add GitHub Actions workflow (after local works)

1. Add `.github/workflows/deploy-mobile.yml`:
   - Trigger: tags `mobile/v*` and `workflow_dispatch`.
   - One job: checkout → Node → install deps → EAS CLI → `eas build --platform all --profile preview --non-interactive` (or `--wait`).
   - Use secret `EXPO_TOKEN` (or `EAS_TOKEN`) only.
2. Document required secret in [SECRETS.md](./SECRETS.md) (EXPO_TOKEN for CI).
3. Test with manual run or tag; confirm builds appear in EAS and are available via internal distribution link.

**Deliverable**: Tag or manual run in GitHub produces iOS and Android builds via EAS Internal; only EXPO_TOKEN required in repo secrets.

### Phase 3 (later): TestFlight for iOS

1. Add submit step: `eas submit --platform ios --latest --non-interactive` after build.
2. Wire Apple credentials (App Store Connect API key or Apple ID + app-specific password). See [SECRETS.md](./SECRETS.md).
3. Optional: separate workflow or profile for “internal only” vs “internal + TestFlight”.

**Deliverable**: Same or separate trigger produces iOS build and submits to TestFlight.

### Phase 4 (optional): Android Play internal track

1. Add Android submit step and Google Play service account secret when needed.

### Phase 5 (optional): Notifications and docs

1. Release checklist: bump version, tag `mobile/v*`, push, wait for EAS build link.
2. Optional: post workflow status to Slack/Discord or GitHub.

---

## 6. Required GitHub Secrets (reference)

### First iteration (EAS Internal only)

| Secret | Used for | When |
|--------|----------|------|
| `EXPO_TOKEN` or `EAS_TOKEN` | EAS CLI auth for build | Always |

### Later (TestFlight / Play submit)

| Secret | Used for | When |
|--------|----------|------|
| Apple: `ASC_KEY_ID`, `ASC_ISSUER_ID`, `ASC_KEY_P8_BASE64` or `APPLE_APP_SPECIFIC_PASSWORD` | iOS submit to TestFlight | Phase 2 |
| `GOOGLE_PLAY_SERVICE_ACCOUNT_JSON` | Android submit to Play internal | Phase 3 |

See [SECRETS.md](./SECRETS.md) for where to create and rotate each.

---

## 7. Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| EAS rate limits | Use tag-based or manual trigger; avoid building on every commit. |
| Stale or broken internal build | Document rollback (re-run from previous tag or use previous build link). |
| Secrets rotation | Document in SECRETS.md; EXPO_TOKEN only for this iteration. |

---

## 8. Acceptance Criteria (first iteration)

- [ ] Workflow runs on tag `mobile/v*` and `workflow_dispatch`.
- [ ] Workflow uses only `EXPO_TOKEN` (or `EAS_TOKEN`); no Apple/Google secrets in repo.
- [ ] One successful run produces EAS builds (e.g. iOS and Android) available via EAS Internal distribution link.
- [ ] Docs list required secret and how to obtain it.

---

## 9. Next Steps After Review

1. **Phase 1**: Document local EAS build steps; add `build:preview` (or similar) script; run one local build and verify EAS Internal link.
2. **Phase 2**: Add `.github/workflows/deploy-mobile.yml`; add GitHub secret `EXPO_TOKEN`; run workflow and verify.
3. Confirm trigger (tag `mobile/v*` + workflow_dispatch) and build profile (`preview`).
4. Schedule TestFlight phase when ready.

---

**Document owner**: Implementation team  
**Review**: Approve this plan before adding or changing workflow files.
