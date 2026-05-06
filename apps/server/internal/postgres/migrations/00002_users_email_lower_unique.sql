-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_unique
    ON users (lower(email))
    WHERE email <> '';

-- +goose Down
DROP INDEX IF EXISTS users_email_lower_unique;
