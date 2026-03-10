# Data Model: User Registration and Settings Flow

**Date**: 2026-03-09  
**Feature**: 005-user-registration-settings  
**Database**: PostgreSQL (extends `specs/001-football-community` schema)

## Relationship to 001

This feature **extends** the existing **User** entity from `specs/001-football-community`. No new tables are required for MVP; only additional columns and API request/response shapes.

## User Entity (Extended)

**Purpose**: Support registration with first name, last name, and optional photo; support Settings “Player Profile” tab and display.

**Existing columns** (from 001): `id`, `email`, `password_hash`, `role`, `display_name`, `avatar_url`, `is_verified`, `is_active`, `created_at`, `updated_at`, `last_login_at`.

**Additions for 005**:

| Column        | Type         | Nullable | Description |
|---------------|--------------|----------|-------------|
| first_name    | VARCHAR(100) | YES*     | Given name; required at registration. *Nullable only if backfilling existing users. |
| last_name     | VARCHAR(100) | YES*     | Family name; required at registration. *Nullable only if backfilling existing users. |
| username      | VARCHAR(100) | YES      | Optional login alias; unique if present. Enables “username or email” login. |

- **avatar_url**: Already present in 001; used as “photo” in this spec.
- **display_name**: Can be derived from `first_name` + `last_name` for new users, or kept as editable display string.

**Migration**: Add `first_name`, `last_name`, `username` to `users`; backfill `display_name` from `first_name || ' ' || last_name` where null; enforce uniqueness on `username` where not null.

**Validation Rules**:

- first_name, last_name: 1–100 characters when provided; required on register.
- username: unique, 1–100 characters; optional; if provided, login may use username or email.
- avatar_url: optional; URL or path to stored asset.

**State**: No new state machine; reuse 001 user lifecycle (active, suspended, etc.).

## Settings-Related Data

- **Following**: Uses existing `user_follows` (or equivalent) from 001; no schema change.
- **Player Profile**: View/edit of `users` (first_name, last_name, avatar_url, display_name).
- **Team Manager**: Uses existing team membership/manager tables from 001; no schema change for 005.
- **Logout**: Stateless; no persistent entity.

## Summary

| Entity / Concept   | Action   | Notes |
|--------------------|----------|--------|
| users              | Extend   | Add first_name, last_name, username |
| user_follows       | Use      | No change |
| Auth / sessions    | Use      | JWT/session per 001 |
