## 007 – Player Profile Research

### Decision: One Player Profile per User

- **Rationale**: Keeps the data model simple and aligns with the “my player” mental model from FIFA-style games. It also simplifies navigation (single entry point, no selector).
- **Alternatives considered**:
  - Multiple profiles per user (e.g., different positions or clubs) – rejected for v1 as it complicates UI and API, and isn’t needed for core engagement.

### Decision: Trait System as Fixed Enum (v1)

- **Rationale**: The spec defines a clear, finite set of traits. Encoding these as a fixed enum in both the API and client keeps queries and validations simple, and avoids needing an admin UI.
- **Alternatives considered**:
  - Fully dynamic traits managed via an admin console – rejected for v1 due to extra surface area and migrations.
  - Free-form tag system – rejected to avoid unbounded taxonomy, duplicates, and abuse.

### Decision: Trait Storage Model

- **Rationale**: Store traits as an array of enum codes on the `player_profiles` table (e.g., `traits text[]`) or a join table (`player_profile_traits`) depending on database conventions.
- **Chosen for v1**: A join table (`player_profile_traits`) with `(player_profile_id, trait_code)`:
  - Plays well with existing SQL tooling.
  - Easy to query for “all profiles with trait X”.
- **Alternatives considered**:
  - JSONB column on profile – simpler but less queryable.

### Decision: Career Teams Representation

- **Rationale**: Career entries are a repeating structure with clear semantics (team name, start year, end year/Present), so they belong in a separate table rather than embedded JSON.
- **Chosen model**:
  - Table `career_teams` (or `player_career_entries`) with:
    - `player_profile_id`
    - `team_name`
    - `start_year`
    - `end_year` (nullable for “Present”)
- **Alternatives considered**:
  - JSON array stored on profile – rejected due to awkward querying and sorting.

### Decision: Photo Upload & Storage

- **Rationale**: The app already expects S3-style media storage. Reusing the same pattern avoids new infra.
- **Chosen approach**:
  - Mobile uploads via existing authenticated API which:
    - Validates MIME type and size (≤ 5 MB).
    - Stores the asset in S3 (or equivalent) under a `player-profiles/` prefix.
    - Persists `photo_url` on the player profile.
- **Alternatives considered**:
  - Direct upload from client to S3 via presigned URL – good for future scalability, but not required for v1 if the API already handles uploads.

### Decision: API Shape for Profile Management

- **Rationale**: Player profiles are always tied to the authenticated user, so the API surface should be “current user–centric”.
- **Chosen endpoints (high level)**:
  - `GET /api/me/player-profile` – fetch current user’s profile (or 404 if none).
  - `POST /api/me/player-profile` – create or upsert basic profile.
  - `PUT /api/me/player-profile` – update profile fields.
  - `PUT /api/me/player-profile/traits` – replace trait set.
  - `POST /api/me/player-profile/photo` – upload/update profile photo.
  - `GET/POST/PUT/DELETE /api/me/player-profile/career-teams` – manage career entries.
- **Alternatives considered**:
  - Generic `/api/player-profiles/{id}` CRUD – more flexible for admin/multi-user views, but not needed for v1.

### Decision: Validation Rules

- **Age**: 13–60 inclusive.
  - **Rationale**: Reasonable playing range; avoids obvious invalid ages.
- **Years**:
  - Start year: 1950–current year + 1.
  - End year: nullable, or ≥ start year and ≤ current year + 1.
  - **Rationale**: Covers realistic football careers while allowing “future” current season.

### Decision: Mobile UX & Navigation

- **Rationale**: Use a dedicated `PlayerProfileScreen` and optional tabs (“Profile”, “Career”, “Stats (future)”) similar to the provided mock.
- **Chosen pattern**:
  - Entry from a profile icon / menu.
  - Within the screen:
    - Section for avatar + basic info.
    - Section for traits (chips + “Add Traits” button + modal).
    - Section for career teams list with “Add Team”.
- **Alternatives considered**:
  - Multi-screen wizard (separate screens for basic info, traits, career) – possible future enhancement; v1 will focus on a single composable screen for simplicity.

