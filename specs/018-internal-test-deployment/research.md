# Phase 0 Research: Internal Test Deployment Pipeline

## Decision 1: Use EAS as single build+submit control plane

- **Decision**: Standardize on Expo EAS Build/Submit/Update for both platforms; avoid separate native CI lane tools (fastlane-only path) for now.
- **Rationale**: Fucci already uses Expo and EAS. A single toolchain reduces release drift, lowers onboarding overhead, and centralizes credential handling.
- **Alternatives considered**:
  - Direct Gradle/Xcode CI builds only: more control, but higher maintenance and native complexity.
  - fastlane-driven store submissions as primary: powerful, but duplicates what EAS Submit already covers for managed workflow.

## Decision 2: Separate release intent via three profiles

- **Decision**: Maintain `development`, `preview`, and `production` profiles in `apps/mobile/eas.json`.
- **Rationale**:
  - `development` supports dev-client workflows.
  - `preview` is the canonical profile for internal test distribution.
  - `production` remains release-safe and can share signing assets with stricter approval.
- **Alternatives considered**:
  - Two-profile model (`dev` + `prod`): too coarse for internal QA cadence.
  - Per-branch custom profiles: flexible but configuration-heavy.

## Decision 3: Push credential lifecycle must be completed before internal submission validation

- **Decision**: Treat FCM and APNs setup as hard blockers before final internal-track validation.
- **Rationale**: Internal builds can install without push, but app readiness for key flows requires notification testing for kick-offs and debate replies.
- **Alternatives considered**:
  - Defer push setup post-submission: speeds first upload, but invalidates core QA scenarios.
  - Use local notification simulation only: insufficient for real-device reliability.

## Decision 4: CI pipeline includes gated submit jobs

- **Decision**: Split CI into build-only jobs (automated) and submit jobs (manual dispatch or protected environment approval).
- **Rationale**: Prevent accidental store uploads while preserving high automation and repeatability.
- **Alternatives considered**:
  - Fully automatic submissions on every main merge: too risky during setup.
  - Fully manual local submissions: inconsistent and non-auditable.

## Decision 5: Credential backup policy is mandatory governance

- **Decision**: Capture and securely escrow Android keystore + aliases/passwords and APNs/Apple metadata in team-owned secret management.
- **Rationale**: Lost signing credentials block updates and force costly recovery.
- **Alternatives considered**:
  - Rely only on EAS remote credentials: convenient but operationally risky without independent backup.
  - Personal-account custody: non-compliant with team continuity.

## Decision 6: OTA strategy uses isolated environment channels

- **Decision**: Map EAS Update channels to isolated environments (`dev`, `staging`, `production`) with branch linkage and explicit runtimeVersion policy.
- **Rationale**: Ensures binary/runtime compatibility and predictable update targeting.
- **Alternatives considered**:
  - Single channel for all builds: high risk of leaking unstable updates.
  - Branch-only targeting without channels: harder operational visibility.

## Decision 7: Dependency sequencing for account setup

- **Decision**: Sequence ownership/enrollment tasks first (Apple enrollment, Google Play org setup, Firebase project ownership) before app/config tasks.
- **Rationale**: Most downstream tasks (keys, app records, package/bundle IDs, submission roles) cannot proceed without account-level permissions.
- **Alternatives considered**:
  - Parallel setup by engineers without owner confirmation: often causes rework and blocked credentials.

## Resolved Clarifications

- **Account model**: Use organization-owned accounts (not personal) for Apple, Google, Firebase, Expo.
- **Release gating**: Build automation always-on; submit automation protected by manual approval.
- **Environment split**: Internal testing uses `preview` build profile mapped to `staging` OTA channel.
