# Spec: Automated Mobile Release (EAS Internal → TestFlight Later)

**Status**: Draft  
**Last revised**: 2026-02

## Summary

The mobile app (Expo / React Native) must support **automated distribution** to testing channels so that testers receive builds without manual upload. This aligns with the constitution’s Release Management requirement for automated deployment pipelines and mobile distribution.

## Goals

1. **First iteration**: Automatically **build** with EAS and distribute via **EAS Internal distribution** when a release is triggered (e.g. tag or manual workflow dispatch). Testers get a link from Expo to install (no App Store or Play Store submit). Supports both iOS and Android from one workflow.
2. **Later iteration**: Add **TestFlight** (iOS) and optionally **Google Play internal** track when ready; that phase will introduce Apple/Google credentials and submit steps.
3. **CI/CD**: Use EAS Build (run **locally** first to validate internal distribution); then add a GitHub Actions workflow so the process is repeatable and auditable.

## Non-Goals (first iteration)

- TestFlight or App Store submit (deferred).
- Google Play submit (deferred).
- App Store / Play Store production release (always out of scope for this spec).

## Success Criteria (first iteration)

- **Local first**: Maintainers can run EAS build locally (e.g. `eas build --profile preview`) and get builds available via **EAS Internal distribution** (install link from Expo dashboard). Steps and scripts are documented.
- **Then CI**: A GitHub Actions workflow can run the same build on tag or manual trigger; only **EXPO_TOKEN** is required; no Apple or Google credentials for this iteration.
- Required secrets and configuration are documented; a new maintainer can run a local build and, when ready, enable the workflow.
- The workflow does not block or slow down the existing API deploy pipeline.

## Constraints

- Use existing Expo EAS setup (`eas.json`, `apps/mobile`) and avoid breaking local development.
- Credentials must be stored as GitHub secrets; no secrets in repo.
- Workflow runs in GitHub Actions; EAS CLI is used for build (submit added in a later phase).

## References

- Constitution: Release Management → Mobile distribution (automated TestFlight / internal testing).
- Existing: `.github/workflows/deploy-api.yml` (API deploy pattern), `apps/mobile/eas.json` (EAS build profiles, including `preview` with `distribution: "internal"`).
