# Implementation Plan: Photo upload (Cloudinary)

**Branch**: `008-photo-upload-cloudinary` | **Date**: 2026-03-24 | **Spec**: [spec.md](./spec.md)

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Enable **photo upload** for **account avatar** and **player profile photo** using **Cloudinary** as CDN/storage. Clients upload images **directly to Cloudinary** using **server-issued signatures** (or a restricted preset), then persist the returned **`secure_url`** via existing or extended API fields (`avatar_url` on user profile, `photo_url` on player profile). The API validates URLs against an allowlist; secrets remain server-side.

### Specification alignment (v1) — [spec.md](./spec.md)

| Requirement | Implication for implementation |
|-------------|--------------------------------|
| **FR-004 / 5 MB** | Client pre-checks file size **≤ 5 MB** before Cloudinary upload; preset and/or API validation match the same cap. |
| **FR-008** | **Player profile photo** is wired only from **Player Profile** after the profile exists; **Create Player Profile** has **no** player photo upload UI. |
| **FR-009** | **Replace-only:** users swap images by uploading again; **no** “remove photo” or **null** URL to clear avatar/player photo in v1 (initial never-set `null` remains valid). |
| **FR-010** | **No** required in-app crop/rotate/filter before upload; circular masks / card framing are **display-only** or Cloudinary **delivery** transforms. |
| **FR-003** | **Library + camera** (Expo ImagePicker); handle permission denial with copy + Settings path. |

## Technical Context

**Language/Version**: Go 1.22+ (API), TypeScript strict (Expo / React Native mobile)  
**Primary Dependencies**: chi HTTP router (existing), Cloudinary signing (Go: HMAC/SHA1 per Cloudinary docs or official SDK), Expo ImagePicker + `fetch`/`FormData` for upload on mobile  
**Storage**: PostgreSQL existing columns `users.avatar_url`, `player_profile.photo_url`; binary blobs not stored in DB  
**Testing**: `go test` for API handlers and URL validation; Jest/React Native tests for client helpers with mocked fetch  
**Target Platform**: iOS/Android via Expo; Linux/macOS for API  
**Project Type**: Mobile + API (monorepo: `apps/mobile`, `services/api`)  
**Performance Goals**: Persist-URL API **< 200ms p95**; Cloudinary upload latency excluded from API SLO; mobile shows loading state until persist completes  
**Constraints**: No Cloudinary **api_secret** in client bundles; HTTPS-only URLs; max image size aligned with Cloudinary preset (e.g. ≤ 5MB)  
**Scale/Scope**: Two surfaces (Settings/avatar + Player Profile); single upload pipeline reused with `context` = `avatar` | `player_profile`

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

**Code Quality Standards:**

- [x] TypeScript strict mode compliance verified
- [x] ESLint configuration defined with zero warnings
- [x] Function complexity ≤ 10, length ≤ 50 lines
- [x] Meaningful naming conventions established

**Testing Standards:**

- [x] TDD approach planned for new features
- [x] Unit test coverage target ≥ 80% identified
- [x] Integration test requirements defined
- [x] E2E test scenarios for P1 user stories planned

**User Experience Consistency:**

- [x] Design system compliance verified
- [x] Accessibility requirements (WCAG 2.1 AA) identified
- [x] Loading states and error handling planned
- [x] Responsive design considerations documented

**Performance Requirements:**

- [x] Performance benchmarks defined (load times, latency)
- [x] Bundle size impact assessed
- [x] Database query performance targets set
- [x] Caching strategy planned

**Developer Experience:**

- [x] Documentation requirements identified
- [x] API documentation needs defined
- [x] Development environment setup documented
- [x] Code review guidelines established

## Project Structure

### Documentation (this feature)

```text
specs/008-photo-upload-cloudinary/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── api.yaml
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
services/api/
├── internal/api/
│   ├── auth.go                    # PUT profile avatar_url validation (extend)
│   ├── player_profile.go          # Accept photo_url on create/update (extend)
│   └── cloudinary.go              # NEW: signature handler + URL validation helpers
├── internal/api/*_test.go
└── main / config for env vars

apps/mobile/src/
├── services/
│   ├── auth.ts                    # updateProfile with avatar_url
│   ├── playerProfile.ts           # pass photo_url after upload
│   └── cloudinaryUpload.ts        # NEW: get signature + upload + return secure_url
├── screens/
│   ├── SettingsScreen.tsx         # avatar picker + upload (or dedicated component)
│   └── PlayerProfileScreen.tsx    # player photo flow (not CreatePlayerProfileScreen per FR-008)
└── components/                    # optional shared ImagePicker + progress UI
```

**Structure Decision**: Implement signing and URL validation in **`services/api`**, shared upload orchestration in **`apps/mobile/src/services/cloudinaryUpload.ts`**, and wire **Settings** + **Player Profile** screens to the shared helper (**not** Create Player Profile per FR-008). No new top-level app; follows existing mobile + API layout.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
| --------- | ---------- | ------------------------------------- |
| Constitution lists **S3** for media; this feature uses **Cloudinary** | Product requirement: CDN + transforms + user-requested vendor; profile images are not stored on raw S3 in v1 | Raw S3 + separate CDN adds operational overhead without matching Cloudinary’s integrated upload + delivery UX |

## Constitution Check (post–Phase 1 design)

- **Code quality**: New handlers split so signing, validation, and HTTP layers stay under complexity/length limits.
- **Testing**: Table-driven tests for URL allowlist; integration tests for signature endpoint with mocked env; mobile unit tests for upload helper with `fetch` mock.
- **UX**: Loading indicators during pick + upload + persist; map `ApiRequestError` to user-facing copy (existing pattern). **No** remove-photo control in v1 (FR-009). **No** crop step before upload (FR-010).
- **Performance**: DB updates are single-row writes by primary key; no N+1.
- **DX**: `quickstart.md` + `contracts/api.yaml` document env vars and paths; OpenAPI for handoff.
