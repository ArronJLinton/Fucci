-- Parseable DDL for sqlc only. Production DBs create these types in
-- sql/schema/20260702120000_create_match_stories.sql (inside DO blocks, which sqlc skips).
-- Keep this definition in sync with that migration.
CREATE TYPE story_scope_type AS ENUM ('match', 'tournament');
CREATE TYPE story_content_type AS ENUM ('photo', 'video');
