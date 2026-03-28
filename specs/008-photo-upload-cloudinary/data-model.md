# Data Model: Photo upload (Cloudinary)

Aligned with [spec.md](./spec.md) **FR-004**, **FR-008–FR-010**.

## Existing fields (no new tables required for MVP)

| Location | Column / field | Type | Purpose |
|----------|------------------|------|---------|
| `users` | `avatar_url` | `TEXT` (nullable) | Account profile image; stores full **Cloudinary HTTPS URL** |
| `player_profile` | `photo_url` | `TEXT` (nullable) | Player card / hero image; stores full **Cloudinary HTTPS URL** |

## Validation rules (application layer)

### Upload size (before Cloudinary)

- **5 MB (5,242,880 bytes) maximum** per file for both avatar and player profile flows (**FR-004**).
- Mobile MUST reject oversize files before starting upload; Cloudinary preset (if used) SHOULD match the same cap.

### `avatar_url` (user)

- Optional; **null** means no avatar ever set, or legacy empty state.
- When set to a non-null value, MUST be `https:` and pass **Cloudinary URL allowlist** (host + optional path prefix).
- **v1 (FR-009):** There is **no** supported transition from “has URL” back to **null** via product/API “clear avatar”; users **replace** by uploading a new image. (Do not expose a remove action.)

### `photo_url` (player profile)

- Optional; **null** means no player photo yet (including after create flow without photo — **FR-008**).
- Same URL rules as `avatar_url`; may use a different **folder** prefix in Cloudinary to separate assets.
- **v1 (FR-009):** **Replace-only** after a URL exists; **no** explicit clear-to-null in API/product for player photo.

### MIME / type

- Must stay **consistent** across client checks, Cloudinary upload constraints, and any API checks (**FR-004**); exact allowlist is defined in implementation (preset + validators).

## Relationships

- **User** 1 — 1 **Player profile** (existing 007 model); photos are independent (user can have avatar without player photo and vice versa).
- **FR-008:** Player photo is added/changed only after a profile exists, from **Player Profile** UI — not during **Create Player Profile**.

## State transitions

1. **Empty → URL**: After successful Cloudinary upload, client sends `secure_url` to API; API validates and updates row.
2. **URL → new URL**: Replace string on new successful upload (**FR-009**); old Cloudinary asset may remain server-side unless a later cleanup job exists.
3. **Never-set (`null`)**: Valid initial state; user may later set a URL via upload + persist.
4. **Clear URL after set:** **Not** in v1 (**FR-009**); deferred if product adds remove/clear later.

## Presentation vs stored asset (**FR-010**)

- Stored value is always the **as-uploaded** Cloudinary resource (subject to size/type rules).
- **Circular avatars**, card masks, and similar effects are **UI layout** and/or **Cloudinary delivery transformation URLs** at read time — not a separate “edited” blob before upload.

## Configuration (environment / secrets)

| Name | Used by | Notes |
|------|---------|------|
| `CLOUDINARY_CLOUD_NAME` | API | Public in upload URL |
| `CLOUDINARY_API_KEY` | API | Used in signing |
| `CLOUDINARY_API_SECRET` | API | Server-only |
| Optional `CLOUDINARY_UPLOAD_PRESET` | API / client | If using unsigned preset path; **max 5 MB** must match **FR-004** |

No new entities required; optional future table `media_assets` if you need deletion tracking or audits.
