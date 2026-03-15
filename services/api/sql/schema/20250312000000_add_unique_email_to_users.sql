-- +goose Up

-- Enforce unique email so duplicate sign-ups return 409 (handler checks pq 23505)
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);

-- +goose Down

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
