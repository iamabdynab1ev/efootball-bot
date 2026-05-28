-- +goose Up
CREATE TABLE IF NOT EXISTS admin_credentials (
    id            SERIAL PRIMARY KEY,
    user_id       INT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    username      TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS admin_credentials;
