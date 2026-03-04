# Required Secrets for Mobile Release (EAS Internal → TestFlight Later)

This document lists the GitHub Actions secrets for the automated mobile release workflow (`.github/workflows/deploy-mobile.yml`) and where to create or obtain each value.

---

## First iteration: EAS Internal distribution only

No App Store or Play Store submit. Only EAS Build is used; builds are distributed via **EAS Internal distribution** (link from Expo dashboard). No Apple or Google credentials required.

### GitHub repository secret

Add under **Settings → Secrets and variables → Actions**. Do not commit the value.

| Secret name   | Used for              | Where to create / obtain |
|---------------|------------------------|---------------------------|
| `EXPO_TOKEN`  | EAS CLI auth for build | [expo.dev](https://expo.dev) → Account → Access tokens. Create a token with scope for your Expo project. Prefer `EXPO_TOKEN` for compatibility with EAS CLI. |

**Rotation**: Revoke in expo.dev and create a new token; update the GitHub secret.

---

## Later: TestFlight (iOS) and Play (Android)

When you add **TestFlight** or **Google Play internal** submit in a future phase, you will need the following. Do not add these until that phase is implemented.

### iOS (TestFlight submit)

Use **one** of:

#### Option A: App Store Connect API key (recommended for CI)

| Secret name        | Where to create / obtain |
|--------------------|---------------------------|
| `ASC_KEY_ID`       | App Store Connect → Users and Access → Keys. Create a key with **App Manager** or **Admin** role. Note the Key ID. |
| `ASC_ISSUER_ID`    | App Store Connect → Users and Access → Keys. Shown at top of the Keys page. |
| `ASC_KEY_P8_BASE64`| After creating the key, download the `.p8` file once. Base64-encode: `base64 -i AuthKey_XXXXXXXX.p8 | tr -d '\n'` and store the result. |

#### Option B: Apple ID + app-specific password

| Secret name                    | Where to create / obtain |
|--------------------------------|---------------------------|
| `APPLE_APP_SPECIFIC_PASSWORD`  | [appleid.apple.com](https://appleid.apple.com) → Sign-In and Security → App-Specific Passwords. |

Apple ID and team/app IDs are usually in `eas.json` or EAS secrets.

### Android (Play internal track, optional)

| Secret name                         | Where to create / obtain |
|-------------------------------------|---------------------------|
| `GOOGLE_PLAY_SERVICE_ACCOUNT_JSON`   | Google Play Console → Setup → API access. Create a service account, grant access to the app, download JSON key. Paste entire JSON as the secret value. |

---

## Summary

- **This iteration**: Only `EXPO_TOKEN` is required. Builds use EAS Internal distribution.
- **Later (TestFlight)**: Add Apple credentials when adding the iOS submit step.
- **Later (Play)**: Add `GOOGLE_PLAY_SERVICE_ACCOUNT_JSON` when adding the Android submit step.
