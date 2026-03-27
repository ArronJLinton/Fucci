# Quickstart: Photo upload (Cloudinary)

## Prerequisites

- Cloudinary account (free tier OK for dev).
- Fucci API and mobile app running per repo README.
- Feature branch `008-photo-upload-cloudinary` checked out.

## 1. Cloudinary dashboard

1. Create a folder structure, e.g. `fucci/avatars` and `fucci/player-profiles`.
2. Configure an **upload preset** (unsigned or signed policy) with:
   - Max file size (e.g. 5 MB)
   - Allowed formats: `jpg`, `png`, `webp`
3. If using **signed uploads**, note API key/secret for server env (never commit secrets).

## 2. API environment

Set in `services/api` runtime (copy from `services/api/.env.example` and fill values):

```bash
CLOUDINARY_CLOUD_NAME=your_cloud_name
CLOUDINARY_API_KEY=...
CLOUDINARY_API_SECRET=...
```

Restart the API. Verify health as usual.

## 3. Verify upload signature endpoint (once implemented)

```bash
# Example payload for avatar context
curl -sS -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"context":"avatar"}' \
  "$API_ORIGIN/v1/api/upload/cloudinary/signature"
```

Expect JSON with fields required for the client to POST to Cloudinary (e.g. `signature`, `timestamp`, `cloud_name`, `folder`).

## 4. Mobile dev flow

1. Sign in.
2. Open Settings (avatar) or Player Profile (photo).
3. Pick an image; confirm upload progress completes.
4. Reload profile: image URLs should be `https://res.cloudinary.com/...`.

## 5. Tests

```bash
cd services/api && go test ./internal/api/... -run Cloudinary  # adjust after tests land
cd apps/mobile && npm test   # or project’s test command
```

## Troubleshooting

| Symptom | Check |
|--------|--------|
| 401 on signature | JWT valid? |
| Cloudinary 401 | Signature/timestamp/params mismatch |
| URL rejected by API | Host not `res.cloudinary.com` or path outside allowed folder |
| Image not showing | HTTPS mixed content, wrong URL field bound in UI |
