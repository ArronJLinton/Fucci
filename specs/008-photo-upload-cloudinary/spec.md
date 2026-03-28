# Feature Specification: Photo upload (Cloudinary)

**Feature Branch**: `008-photo-upload-cloudinary`  
**Created**: 2026-03-24  
**Status**: Draft  
**Input**: User description: "We need to create photo upload functionality. I want to use the Cloudinary CDN to achieve this. At the moment, a photo upload operation is necessary for the profile photo and player profile."

## Clarifications

### Session 2026-03-24

- Q: Is player profile photo available on the initial Create Player Profile flow, or only after the profile exists? → A: **Edit-only (v1):** upload player photo from Player Profile after creation; **not** on the Create Player Profile screen.
- Q: In v1, can users explicitly **remove** (clear) account avatar and/or player photo, or only **replace** with a new upload? → A: **Replace-only (v1):** users may upload a new image to replace an existing one; there is **no** explicit remove/clear action for account avatar or player photo in v1.
- Q: What is the maximum upload size per image for avatar and player profile photos? → A: **5 MB** maximum per image (client and server-aligned validation).
- Q: For picking images on mobile, is **library only**, **camera only**, or **both** required in v1? → A: **Library + camera** — user can choose an existing photo or capture a new one (same for account avatar and player profile photo).
- Q: Must v1 include **in-app cropping or other editing** before upload? → A: **No** — upload **as-is** after library/camera selection; no required crop/rotate/filter step in v1 (presentation may still use circular masks or Cloudinary delivery transforms at display time).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Account profile photo (Priority: P1)

As a signed-in user, I want to choose a photo from my device (or camera) and set it as my **account profile picture** so that it appears wherever my identity is shown (e.g. debates, comments, settings).

**Why this priority**: Account avatar is the primary identity surface across the app; it is required for a complete profile experience.

**Independent Test**: Upload succeeds, `avatar_url` in user profile reflects a Cloudinary HTTPS URL, and the image renders in Settings / debate UI.

**Acceptance Scenarios**:

1. **Given** I am logged in and open account / settings where avatar is editable, **When** I choose a supported image from the **library** or **camera** and confirm, **Then** the app shows progress and my new avatar appears after upload completes.
2. **Given** I upload an invalid file type or a file **over 5 MB**, **When** the client or API validates, **Then** I see a clear error and my previous avatar is unchanged.

---

### User Story 2 - Player profile photo (Priority: P1)

As a user who **already has a player profile**, I want to upload a **player profile photo** from the **Player Profile** screen (from **library** or **camera**) so it displays on the FIFA-style player profile hero/card.

**Why this priority**: Player profile is a core 007 feature; photo is part of the fantasy-card experience.

**Independent Test**: After upload, `GET /player-profile` returns `photo_url` pointing to Cloudinary; the mobile player profile screen shows the image.

**Acceptance Scenarios**:

1. **Given** I have a player profile, **When** I upload a player photo from Player Profile (edit) via library or camera, **Then** the hero/card image updates and persists across sessions.
2. **Given** I just finished Create Player Profile without a photo, **When** I open Player Profile and upload a photo, **Then** the same persistence and display rules apply as for any later edit.

---

### User Story 3 - Replace photos (Priority: P2)

As a user, I want to **replace** my account avatar or player profile photo with a **new** image later so I can update my look without contacting support.

**Why this priority**: Standard profile hygiene; lower than first-time upload.

**Independent Test**: After a successful second upload, `avatar_url` or `photo_url` reflects the new Cloudinary URL end-to-end.

**Acceptance Scenarios**:

1. **Given** I already have an avatar or player photo, **When** I upload a new image, **Then** the new Cloudinary URL is stored and displayed (replacing the previous URL).
2. **Given** I have never set a player photo (`photo_url` null), **When** I upload a first image, **Then** the URL is stored and shown; **v1 does not require** an explicit “remove photo” path afterward.

---

### Edge Cases

- **Create Player Profile flow** does **not** include player photo upload in v1; users add the photo afterward on Player Profile.
- Network loss mid-upload: user sees retry-friendly messaging; no partial URL stored unless upload to Cloudinary completed successfully.
- Expired or invalid auth during upload: user is prompted to sign in again (401/403).
- Cloudinary or API outage: user sees a non-technical error with retry guidance.
- Malicious URL: server validates that stored URLs belong to allowed Cloudinary host/folder patterns before persisting.
- **Replace-only (v1):** No UI or API requirement to set `avatar_url` or `photo_url` to **null** to “clear” an image after one was set; users replace by uploading again. Initial **null** (never uploaded) remains valid.
- **Oversized file:** Selection or upload of a file **larger than 5 MB** MUST be rejected before or at Cloudinary upload with user-visible guidance.
- **Permissions:** If the user **denies** camera or photo library permission, the app MUST explain what is needed and how to retry (e.g. Settings); no silent failure.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST store profile images on **Cloudinary** and persist **HTTPS URLs** in application data (`users.avatar_url`, `player_profile.photo_url`).
- **FR-002**: The system MUST NOT expose **Cloudinary API secrets** to client builds; secrets stay on the server or use **signed/unsigned upload presets** configured per security review.
- **FR-003**: Mobile clients MUST let the user choose an image from the **photo library** or **camera** (both flows) for account avatar and player profile photo, with clear loading and error states during upload and persistence.
- **FR-004**: The API and mobile client MUST enforce a **5 MB** maximum per image (same limit everywhere) and **consistent** allowed MIME/types with Cloudinary upload rules and API checks; rejection MUST occur before persisting a URL or via matching preset limits.
- **FR-005**: Account avatar updates MUST integrate with existing `PUT /users/profile` (or equivalent) `avatar_url` field.
- **FR-006**: Player profile photo MUST be set via API after upload (e.g. `photo_url` on profile update or dedicated endpoint) and MUST align with OpenAPI under `/v1/api`.
- **FR-007**: Automated tests MUST cover signature/upload-config logic and URL persistence (unit + integration per constitution).
- **FR-008**: In v1, **player profile photo** MUST be addable or changeable only from the **Player Profile** experience after the profile exists; the **Create Player Profile** screen MUST NOT include player photo upload.
- **FR-009**: In v1, the product MUST support **replacing** an existing avatar or player photo by uploading a new image; it MUST **not** expose an explicit **remove/clear photo** action for either surface (defer to a later release if needed).
- **FR-010**: In v1, the mobile app MUST **not** require **in-app cropping, rotation, or filters** before upload; the file uploaded to Cloudinary is the user’s library/camera result (subject to size/type rules). UI may still show **circular or framed** images using layout or CDN **delivery** transforms only.

### Key Entities

- **User (account)**: `avatar_url` — CDN URL for account picture.
- **Player profile**: `photo_url` — CDN URL for in-game style card photo.
- **Upload session (logical)**: ephemeral; client uploads to Cloudinary, then sends resulting URL to API.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% of successful Cloudinary uploads result in a persisted URL visible on the next GET of user/player profile within 5 seconds.
- **SC-002**: P95 API time to persist URL after client provides it is under 200ms (constitution API target), excluding Cloudinary upload time.
- **SC-003**: Users receive actionable error copy on validation, auth, and network failures (no silent failures).
