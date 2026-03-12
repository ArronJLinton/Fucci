# Research: User Registration and Settings Flow

**Feature**: 005-user-registration-settings  
**Date**: 2026-03-09

## 1. Registration Fields and Existing User Model

**Decision**: Extend API and, if needed, DB to support **first_name**, **last_name**, and **photo (avatar_url)** in addition to existing email/password/display_name. Treat “username or email” as a single login identifier (email when valid email format, else username).

**Rationale**: Spec requires first name, last name, and optional photo. 001 schema has `display_name` and `avatar_url`; it does not have separate first/last. To meet the spec without breaking existing clients, add first_name and last_name (nullable initially or backfilled from display_name) and keep avatar_url for photo.

**Alternatives considered**: (a) Use only display_name — rejected because spec explicitly requires first and last name. (b) Store “username” separately from email — optional future enhancement; for MVP a single identifier field (email or username) is sufficient.

---

## 2. Password Rules and Security

**Decision**: Enforce minimum length (e.g. 8 characters), server-side hash (bcrypt or argon2), and optional complexity (e.g. one letter, one number). Document in API contract and show same rules in mobile UI.

**Rationale**: Constitution and security best practices require secure password handling; 001 already specifies password_hash. Aligning client validation with server reduces invalid submissions and improves UX.

**Alternatives considered**: No client-side rules — rejected because it increases round-trips and frustrates users; server remains source of truth.

---

## 3. Settings Tabs and Navigation

**Decision**: Single **Settings** screen with three content tabs: **Following**, **Player Profile**, **Team Manager**. **Logout** is a single action (button or list item) at the bottom of the screen, not a tab. Tab order: Following | Player Profile | Team Manager.

**Rationale**: Matches attached UI (tabs for Following, Player Profile, Team Manager; Logout as bottom action). Keeps one screen to implement and one place for “edit profile” entry (Player Profile tab).

**Alternatives considered**: Separate screens per section — rejected for MVP to match provided design and reduce navigation depth.

---

## 4. Session Storage (no \"Remember me\" toggle)

**Decision**: Sessions are always persisted securely using platform secure storage (e.g. Expo SecureStore / Keychain / Keystore). There is no explicit “Remember me” toggle in the UI for this phase.

**Rationale**: Simpler UX and implementation; avoids confusion when the toggle is not wired differently from the default behavior. Users can still log out explicitly from Settings.

**Alternatives considered**: Adding a “Remember me” toggle that controls persistence; deferred to a future spec if needed for shared-device scenarios.

---

## 5. Forgot Password Scope

**Decision**: Include “Forgot password?” as a UI entry point (link on Login screen). Actual reset flow (email send, token, new password) can be Phase 2 if not already implemented in 001; API contract should reserve endpoint (e.g. POST /auth/forgot-password, POST /auth/reset-password).

**Rationale**: Spec calls out Forgot password; UX expects the link; backend implementation can follow existing 001 or a later task.

**Alternatives considered**: No forgot password — rejected because it is a standard expectation and called out in the spec.

---

## 6. Social Login (Google / Apple / Facebook)

**Decision**: Out of scope for this spec’s implementation. UI may show “Continue with Google/Apple/Facebook” as placeholders or disabled; actual OAuth/OpenID integration is a separate spec/task.

**Rationale**: Spec states scope is email/username + password; social providers to be added later.

**Alternatives considered**: Implement one provider (e.g. Apple) — deferred to keep MVP focused on email/password and settings structure.

---

## 7. Team Manager Tab Visibility

**Decision**: Show **Team Manager** tab to all users; for users without team_manager (or admin) role, show an empty state or “Request access” / “Upgrade” message rather than hiding the tab. Alternatively, hide the tab when user is not team_manager if product prefers.

**Rationale**: Spec says “for team_manager role”; either visible-with-message or hidden both satisfy “support Team Manager”; product can choose. Document in tasks.

**Alternatives considered**: Always hide for non–team-managers — acceptable alternative; document in implementation tasks.
