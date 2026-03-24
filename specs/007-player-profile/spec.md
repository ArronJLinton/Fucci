## 007 – Player Profile Experience

### Overview

Build a **Player Profile** experience inspired by FIFA / EA Sports for football (soccer) fans.  
Users can create and manage a player profile with core attributes (age, country, club/free agent, position), attach **traits**, upload a **profile photo**, and maintain a **career history** of teams and dates.

The experience spans:

- **Mobile app (React Native / Expo)** – primary UI for creating and editing profiles.
- **Existing Go API** – new endpoints for persisting player profiles, traits, photos, and career teams.

### Clarifications

#### Session 2026-02-15

- Q: Can the user delete their player profile and create a new one (full reset), or is it edit-only? → A: Delete allowed.
- Q: How should the Country field be implemented if no existing country picker component exists? → A: Create a reusable searchable country picker (ISO codes, optional flags).

### Goals

- Allow an authenticated user to:
  - Create a **single player profile** linked to their account (one profile per account).
  - **Delete** their profile and create a new one from scratch (full reset).
  - Edit core attributes: age, country, club/free agent, position.
  - Select from a curated list of **player traits**.
  - Upload and display a **profile photo**.
  - Add and edit **career teams** with date ranges.
- Deliver a UI that feels close to the attached FIFA-style reference:
  - Clean card layout, traits rendered as “chips” / icons.
  - Clear separation between **Profile**, **Stats (future)**, and **Career** sections.

### Non‑Goals (for this spec)

- No in‑depth player statistics (goals, assists, ratings).
- No multi‑profile support per user.
- No matchmaking or team recruitment features.
- No public discovery / search of player profiles (v1 is “my profile” only).

### Primary User Story

> As a football fan, I want to create a player profile that captures my basic details, traits, and career history so I can represent myself like a real-world player.

### Detailed Requirements

#### 1. Basic Profile Creation

On first entry to the Player Profile flow:

- If the user has **no profile**, show a **Create Player Profile** screen:
  - **Inputs (required unless noted):**
    - **Age** – integer, 13–60 inclusive.
    - **Country** – required; use a reusable searchable country picker component backed by ISO country codes (with flags where available).
    - **Club or Free Agent** – text input with optional “Free Agent” toggle.
    - **Position** – required; single select from a fixed enum:
      - `GK`, `DEF`, `MID`, `FWD` (later expandable to more granular positions).
  - Buttons:
    - **Next** (primary) – validates fields, creates/updates profile draft via API.
    - **Or Maybe Later** (secondary) – dismisses flow without creating a profile.

#### 2. Player Traits

After basic profile creation, users can **add and edit traits**:

- Supported traits (enum, stored as codes):
  - Leadership
  - Finesse Shot
  - Playmaker
  - Speed Dribbler
  - Long Shot Taker
  - Outside Foot Shot
  - Power Header
  - Flair
  - Power Free Kick
- UX:
  - “Add Traits” button opens a **full-screen modal** similar to the attached reference:
    - Title: **Select Player Traits**.
    - Each trait row shows an icon, name, and a checkmark when selected.
    - Multi-select; up to **5 traits** can be active at once (hard limit).
    - Actions:
      - **Save** (primary) – persists selected traits to the profile.
      - **Back / Close** – dismisses without saving changes.
  - On the profile screen, traits are rendered as pill chips / badges in a horizontal wrap layout.

Behavior:

- Traits are stored server-side as an array of trait codes.
- When saving traits, the entire set is replaced with the client’s current selection.

#### 3. Profile Photo Upload

- Users can upload a **single profile photo**:
  - Source: device image picker (camera roll; camera capture is a stretch goal).
  - Accepted formats: JPEG, PNG; max size 5 MB.
- UX:
  - Circular avatar placeholder with **“Upload Photo”** CTA (see reference).
  - After upload:
    - Show the new photo in the avatar.
    - Show loading state while upload is in progress.
    - Show error toast/banner on failure with retry option.

Backend:

- API returns a **public URL** for the stored image (e.g., S3-presigned path).
- Player profile stores `photo_url` as part of the profile record.

#### 4. Career Teams

Users can add a chronological list of teams they have played for:

- Each **career team entry** includes:
  - **Team Name** – required string.
  - **Start Year** – required integer (e.g., 2018).
  - **End Year** – optional integer; may be “Present”.
- UX:
  - Career section below traits:
    - List of existing entries:
      - `Team Name` + years (e.g., “Juventus FC — 2018–2020”).
    - “Add Team” / “Add Career Team” button opens an inline form or modal:
      - Inputs: team name, start year, end year, “Present” checkbox.
      - Validations:
        - Start year ≤ end year when both provided.
        - Years must be within a sensible range (e.g., 1950–current year + 1).
    - Ability to **edit** and **delete** existing entries.

Behavior:

- Teams are stored as a list of career entries associated with the player profile.
- Order is **descending by start year** (most recent first) in the UI.

#### 5. Navigation & Access

- Entry point:
  - From the app’s main nav (e.g., profile icon or “Player Profile” menu item).
- Authentication:
  - Only authenticated users can create/edit a profile.
  - Unauthenticated users attempting to access the profile flow are redirected to login/signup.

#### 6. Profile lifecycle

- Users may **delete** their player profile. After deletion, the user has no profile and will see the **Create Player Profile** flow again on next entry; they may create a new profile from scratch.
- Delete is a destructive, one-step action (confirm before delete). No soft delete or “restore” in v1.

### Technical Assumptions

- Mobile app: existing **React Native + Expo** client.
- Backend: existing **Go / PostgreSQL** API.
- Image storage: existing S3-compatible bucket (or similar) used elsewhere in the app.
- One profile per authenticated user (no multi-profile support).

### Open Questions

- Should traits be fully configurable from an admin panel or hard-coded for now?  
  → **v1 decision**: hard-code trait enum in both client and server, with a simple metadata endpoint for future flexibility.
- Should player profiles be publicly shareable via link?  
  → **Out of scope** for v1.

