-- Parseable DDL for sqlc only. Production DBs create this type in
-- sql/schema/016_add_auth_to_users.sql (inside a DO block, which sqlc skips).
-- Keep this definition in sync with that migration.
CREATE TYPE user_role AS ENUM ('fan', 'team_manager', 'admin');
