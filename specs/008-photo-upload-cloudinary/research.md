# Phase 0 Research: Cloudinary photo upload

## 1. Upload model: direct to Cloudinary vs via API

**Decision**: **Client uploads directly to Cloudinary** (HTTPS `POST` to `https://api.cloudinary.com/v1_1/{cloud_name}/image/upload`), then the client calls the **Fucci API** with the returned `secure_url` to persist `avatar_url` or `photo_url`.

**Rationale**:

- Avoids large multipart bodies on the Go API and keeps API latency low for the “persist URL” step (aligns with constitution **< 200ms p95** for standard operations).
- Cloudinary is optimized for ingest and CDN delivery; API only validates and stores strings.

**Alternatives considered**:

- **Server proxies file to Cloudinary**: Simpler client but doubles bandwidth on API, higher latency, larger payloads to monitor.
- **Presigned S3 + Cloudinary fetch**: More moving parts; Cloudinary-only flow is sufficient for this feature.

---

## 2. Authentication to Cloudinary: signed vs unsigned upload

**Decision**: **Server-generated upload signature** (or **restricted unsigned upload preset**) issued behind auth.

- **Recommended for production**: `POST /v1/api/.../cloudinary/signature` (authenticated) returns `timestamp`, `signature`, `api_key`, `cloud_name`, `folder`, and optional `public_id` prefix so the client can call Cloudinary upload with **signed** parameters. API secret never leaves the server.

**Rationale**:

- Unsigned presets are easy but widen abuse surface if the preset leaks; signing ties each upload to server policy.

**Alternatives considered**:

- **Unsigned upload preset** only: acceptable for MVP if preset is locked to a single folder, max file size, and allowed formats in the Cloudinary dashboard; still expose only `cloud_name` + `upload_preset` from a small config endpoint.

---

## 3. URL validation before DB write

**Decision**: API validates `avatar_url` / `photo_url` with:

- HTTPS scheme
- Hostname matching `res.cloudinary.com` (or custom CNAME if configured later)
- Optional path prefix match for tenant folder (e.g. `fucci/avatars/{userID}/`, `fucci/player-profiles/{profileID}/`)

**Rationale**: Prevents arbitrary URL injection (tracking pixels, malicious hosts).

**Alternatives considered**: Trust client entirely — rejected for security.

---

## 4. Relationship to constitution “S3 for media storage”

**Decision**: Treat **Cloudinary as the media CDN and storage** for user-generated profile images for this product. Document as a **justified deviation** from the generic “S3” line in the constitution (see `plan.md` Complexity Tracking).

**Rationale**: User explicitly requested Cloudinary; it provides hosting + transforms + CDN in one product.

**Alternatives considered**: S3 + CloudFront + image worker — more ops burden for the same MVP.

---

## 5. Mobile implementation notes

**Decision**: Use **Expo ImagePicker** (or equivalent) for library access; upload with `fetch`/`FormData` to Cloudinary per their multipart upload docs; then `makeAuthRequest` to persist URL.

**Rationale**: Matches existing React Native / Expo stack in `apps/mobile`.

---

## 6. Legacy `/player-profile/photo` multipart endpoint (007 contract)

**Decision**: **Deprecate or replace** multipart binary upload on the API with the **URL persistence** flow, unless the team keeps multipart as a fallback that uploads server-side to Cloudinary. Prefer one clear path: **direct Cloudinary + JSON URL update** to reduce duplication.

**Rationale**: Single code path for storage and validation; multipart can remain as optional Phase 2 if needed.
