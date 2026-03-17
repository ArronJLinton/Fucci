## 007 – Player Profile Data Model

### Entities

#### `player_profiles`

- **Purpose**: Stores the core player profile for a user (one per user).
- **Fields**:
  - `id` (PK, int)
  - `user_id` (FK → `users.id`, unique)
  - `age` (int, nullable until first save; validation: 13–60)
  - `country_code` (string, ISO 3166-1 alpha-2)
  - `club_name` (string, nullable)
  - `is_free_agent` (bool, default false)
  - `position` (enum: `GK`, `DEF`, `MID`, `FWD`)
  - `photo_url` (string, nullable)
  - `created_at`, `updated_at` (timestamps)

#### `player_profile_traits`

- **Purpose**: Join table linking a profile to its selected traits.
- **Fields**:
  - `id` (PK, int)
  - `player_profile_id` (FK → `player_profiles.id`)
  - `trait_code` (enum/string)
    - Allowed values:
      - `LEADERSHIP`
      - `FINESSE_SHOT`
      - `PLAYMAKER`
      - `SPEED_DRIBBLER`
      - `LONG_SHOT_TAKER`
      - `OUTSIDE_FOOT_SHOT`
      - `POWER_HEADER`
      - `FLAIR`
      - `POWER_FREE_KICK`
- **Constraints**:
  - Unique (`player_profile_id`, `trait_code`).
  - Max 5 traits per profile enforced in application layer.

#### `player_career_teams`

- **Purpose**: Stores the list of teams a player has played for.
- **Fields**:
  - `id` (PK, int)
  - `player_profile_id` (FK → `player_profiles.id`)
  - `team_name` (string, required)
  - `start_year` (int, required; 1950–current_year+1)
  - `end_year` (int, nullable; must be ≥ start_year when provided)
  - `created_at`, `updated_at` (timestamps)

### Relationships

- `users 1 — 1 player_profiles`
- `player_profiles 1 — many player_profile_traits`
- `player_profiles 1 — many player_career_teams`

### API-Facing Shapes

#### PlayerProfile DTO

```ts
type PlayerProfile = {
  id: number;
  age: number | null;
  country: string;
  club: string | null;
  is_free_agent: boolean;
  position: 'GK' | 'DEF' | 'MID' | 'FWD';
  photo_url: string | null;
  traits: string[]; // trait_code values
  career_teams: Array<{
    id: number;
    team_name: string;
    start_year: number;
    end_year: number | null;
  }>;
};
```

