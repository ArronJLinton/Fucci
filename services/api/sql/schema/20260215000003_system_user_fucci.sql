-- +goose Up

-- Ensure system user (Fucci) exists for seeded debate comments. One row, identifiable by email.
-- Safe to run at deploy: inserts only if not present (ON CONFLICT DO NOTHING).
INSERT INTO users (firstname, lastname, email, display_name)
VALUES ('Fucci', 'System', 'contact@magistri.dev', 'Fucci')
ON CONFLICT (email) DO NOTHING;

-- +goose Down

-- Remove system user only if it is the Fucci system user (do not remove real users)
DELETE FROM users WHERE email = 'contact@magistri.dev';
