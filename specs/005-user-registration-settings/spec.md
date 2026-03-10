# Feature Specification: User Registration and Settings Flow

**Feature Branch**: `005-user-registration-settings`  
**Created**: 2026-03-09  
**Status**: Draft  
**Input**: User registration and settings flow with Sign up, Login, and Settings (Following, Player Profile, Team Manager, Logout). UI inspiration from attached FUCCI flows and auth screens.

## Scope

This spec covers the **mobile UI and API contract** for:

1. **Sign up / Register** – New account creation with required and optional fields.
2. **Login** – Returning user authentication (email/username + password, optional “Remember me”, forgot password).
3. **Settings** – Single settings screen with tabs: **Following**, **Player Profile**, **Team Manager**, and **Logout**.

Backend auth and user APIs already exist under `specs/001-football-community` (e.g. `POST /auth/register`, `POST /auth/login`, `GET/PUT /users/me`). This spec defines the **registration fields**, **UI flow**, and **settings structure** so implementation stays consistent across client and API.

## Clarifications

- **Username vs email**: Registration accepts either **username or email** as one required field (single input; backend may treat as email when format matches).
- **Names**: **First name** and **Last name** are required at sign-up; profile may show display name (e.g. "First Last" or editable display_name).
- **Photo**: **Photo (avatar)** is optional at sign-up; user can add or change it in profile/settings.
- **Settings tabs**: **Following** = manage followed leagues and teams (toggles); matches are discoverable elsewhere. **Player Profile** = user’s own profile (name, photo, expertise/skills if applicable); **Team Manager** = team management for users with team_manager role; **Logout** = sign out (no separate tab, action at bottom of settings).
- **Social login**: UI may show “Continue with Google / Apple / Facebook”; scope of this spec is **email/username + password** flow; social providers can be added later via separate spec.
- **Session 2026-02-15**: Q: After successful sign-up, what should happen? → A: Automatically sign the user in and navigate to the main app (e.g. Home or Profile).
- **Session 2026-02-15**: Q: When the user taps Logout, should the app ask for confirmation or sign out immediately? → A: Show a confirmation dialog (e.g. "Log out?" with Cancel / Log out); sign out only after confirm.
- **Session 2026-02-15**: Q: For this spec/phase, should "Forgot password?" be a full flow or only a UI entry point? → A: UI only for this phase: "Forgot password?" link and a placeholder screen or "Coming soon"; full flow in a later phase.
- **Session 2026-02-15**: Q: What should the Following tab show and let the user manage? → A: Leagues and teams with toggles; matches are discoverable elsewhere, not in Following.
- **Session 2026-02-15**: Q: What password rules should the spec require? → A: Minimum length (e.g. 8 characters) plus at least one letter and one number (or similar simple rule).

---

## User Scenarios & Testing _(mandatory)_

### User Story 1 – Sign Up (Priority: P1)

A new user can create an account from the app with email/username, password, first name, last name, and optional photo.

**Why this priority**: Registration is the entry point for authenticated features (debates, following, profile).

**Independent Test**: Can be tested by completing the sign-up form, submitting, and then logging in with the same credentials.

**Acceptance Scenarios**:

1. **Given** the app shows the Sign Up screen, **When** the user enters email/username (required), password (required), first name (required), last name (required), and optionally a photo, **Then** they can submit and an account is created.
2. **Given** the user submits with missing required fields, **When** validation runs, **Then** errors are shown and submission is blocked.
3. **Given** the user has an account, **When** they tap “Already have an account? Login”, **Then** they are taken to the Login screen.
4. **Given** sign-up succeeds, **When** the API returns success, **Then** the user is automatically signed in and navigated to the main app (e.g. Home or Profile).

---

### User Story 2 – Login (Priority: P1)

A returning user can sign in with email/username and password, with optional “Remember me” and “Forgot password” flow.

**Why this priority**: Login is required to access profile, settings, and protected features.

**Independent Test**: Can be tested by entering valid credentials and reaching the main app (e.g. Home or Profile).

**Acceptance Scenarios**:

1. **Given** the app shows the Login screen, **When** the user enters email/username and password and taps Login, **Then** they are authenticated and taken to the main app.
2. **Given** credentials are invalid, **When** the user taps Login, **Then** an error message is shown and the user can retry.
3. **Given** the user taps “Forgot password?”, **When** the flow is triggered, **Then** they see a placeholder screen or "Coming soon" message; full reset flow is out of scope for this phase.
4. **Given** the user has no account, **When** they tap “Don’t have an account? Sign Up”, **Then** they are taken to the Sign Up screen.

---

### User Story 3 – Settings and Profile (Priority: P2)

An authenticated user can open Settings, switch between tabs (Following, Player Profile, Team Manager), and log out.

**Why this priority**: Settings centralize account and preferences; Logout is required for security and multi-user devices.

**Independent Test**: Can be tested by logging in, opening Settings from Profile/bottom nav, switching tabs, and logging out.

**Acceptance Scenarios**:

1. **Given** the user is logged in, **When** they open Settings (e.g. from Profile or bottom nav), **Then** they see their profile summary (avatar, name, email) and tabs: **Following**, **Player Profile**, **Team Manager**.
2. **Given** the user is on Settings, **When** they select **Following**, **Then** they see and manage followed leagues and teams via toggles; matches are not in this tab.
3. **Given** the user selects **Player Profile**, **When** the tab is shown, **Then** they can view/edit first name, last name, photo, and any player-specific fields (e.g. skill expertise).
4. **Given** the user has the team_manager role, **When** they select **Team Manager**, **Then** they see team management actions; otherwise the tab may be hidden or show an upgrade/request message.
5. **Given** the user taps **Logout**, **When** a confirmation dialog is shown and the user confirms, **Then** they are signed out and returned to Login or anonymous Home.

---

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST allow users to register with: username or email (required), password (required), first name (required), last name (required), photo (optional).
- **FR-002**: System MUST allow users to log in with username or email and password; support optional “Remember me” and “Forgot password” entry points.
- **FR-003**: System MUST expose a Settings flow with at least: **Following** (manage followed leagues and teams via toggles), **Player Profile** (view/edit name, photo, profile data), **Team Manager** (for team_manager role), and **Logout**.
- **FR-004**: System MUST persist and return first name, last name, and avatar/photo for the user profile; API and DB MUST support these fields (extend existing user model if needed).
- **FR-005**: System MUST validate required registration fields on client and server and return clear validation errors.
- **FR-006**: System MUST use secure password handling (hash on server; no plaintext storage); password rules (length, complexity) MUST be documented and enforced.

### Key Entities

- **User**: Identified by id; has email or username (login identifier), password_hash, first_name, last_name, optional avatar_url/photo, role (fan | team_manager | admin). Aligns with `specs/001-football-community` User entity; this spec adds explicit first_name, last_name, and photo for registration/settings.
- **User profile (settings)**: Same user record; **Following** is a view over user_follows (leagues and teams only); **Team Manager** over team-management APIs; **Player Profile** is the editable subset of user (name, photo, optional expertise/skills).

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: User can complete sign-up (all required fields) in under 2 minutes.
- **SC-002**: User can log in and reach Settings in under 30 seconds with valid credentials.
- **SC-003**: Settings tabs (Following, Player Profile, Team Manager) load within 1s; Logout completes within 2s.
- **SC-004**: Validation errors for sign-up and login are shown inline and are actionable (user knows what to fix).

---

## UI Flow Summary (from attached designs)

- **Sign Up**: FUCCI logo; optional profile photo placeholder; fields: username/email, password, first name, last name; “Sign Up” primary button; “Already have an account? Login”; optional “Or continue with” Google/Apple (and Facebook on Login).
- **Login**: Back arrow; “Login”; username/email, password; “Remember me”; “Login” button; “Forgot password?”; “Don’t have an account? Sign Up”; social buttons.
- **Settings**: Back arrow; user avatar and name/email with chevron (edit profile); tabs: **Following** (active), **Player Profile**, **Team Manager**; under Following: list of leagues and teams with toggles; **Logout** at bottom. Dark theme with blue accents; bottom nav: Home, Matches, News, Profile/Settings.
