# Feature Specification: Internal Test Deployment Pipeline (iOS + Android)

**Feature Branch**: `018-internal-test-deployment`  
**Created**: 2026-04-06  
**Status**: Draft  
**Input**: User description:
"You are a senior mobile engineer helping deploy a React Native (Expo) sports app called Fucci to internal test tracks on both Android and iOS..."

## Summary

Define a complete, dependency-aware deployment task list and design artifacts to ship the Expo mobile app to internal test distribution on both Android (Google Play internal testing) and iOS (TestFlight internal testing), including credentials, push setup (FCM/APNs via Expo), OTA updates, and CI/CD automation with GitHub Actions.

## Clarifications

### Session 2026-04-06

- Q: What ownership model should be used for Apple/Google/Firebase/Expo/GitHub release accounts? → A: Fully organization-owned accounts.
- Q: What CI submission policy should internal deployment use? → A: Build automatically; submit only with manual approval.
- Q: What OTA environment/channel split should be used? → A: Dev, Staging, Production.

## Goals

- Enable reliable internal test releases for Android and iOS from a single repo.
- Establish reproducible credential and secret management for EAS Build/Submit.
- Ensure push notifications function in internal builds for both platforms.
- Provide automated CI workflows for build and submission, with manual approvals where required.
- Document prerequisites and blocking dependencies so release setup can be executed in correct order.

## Non-Goals

- Public production rollout to external users.
- Store listing optimization/content strategy.
- Feature implementation for app flows themselves (onboarding/feed/debates/etc).
- Re-architecture of backend services beyond required deployment integration.

## Stakeholders

- Mobile engineering
- DevOps/Platform
- QA/Internal testers
- Product/Operations

## Functional Requirements

1. Provide a full task inventory for Android internal testing setup in Google Play Console.
2. Provide a full task inventory for iOS internal testing setup in Apple Developer + App Store Connect.
3. Define Firebase/FCM setup required for Expo push on Android.
4. Define APNs key setup required for Expo push on iOS.
5. Define EAS project and `eas.json` profile strategy (`development`, `preview`, `production`).
6. Define credential handling tasks, including keystore backup and Apple provisioning handling.
7. Define OTA strategy via EAS Update and channel/branch mapping.
8. Define GitHub Actions jobs for automated EAS builds and submissions.
9. Mark task dependencies and blockers explicitly.
10. Provide practical validation steps for both internal test tracks after deployment.

## Success Criteria

- A new engineer can execute the task list end-to-end and produce installable internal builds on both platforms.
- Internal testers receive builds via Play Internal Testing and TestFlight.
- Push notifications can be sent and received from internal builds on both platforms.
- CI workflows can build and submit artifacts without local credentials.

## Constraints

- Mobile stack: Expo (React Native), EAS Build/Submit/Update.
- Backend stack: Fly.io + Supabase (Postgres/Realtime/Storage).
- Push provider: Expo push service over FCM/APNs.
- Must use secure secret handling in GitHub and Expo environments.
- Release tooling accounts (Apple, Google, Firebase, Expo, GitHub) must be organization-owned, not personal.
- CI must auto-build, but store/TestFlight submission must remain manually gated (workflow dispatch and/or protected environment approval).
- OTA updates must use three isolated channels/environments: `dev`, `staging`, and `production`.

## Risks

- Delays in Apple enrollment or app record approvals.
- Misconfigured FCM/APNs keys causing push failures.
- Credential loss (Android keystore / Apple signing keys).
- CI secret drift between GitHub and Expo.

## Open Questions

- None.
