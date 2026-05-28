-- +goose Up
-- Делаем telegram_id необязательным (для web-only пользователей)
ALTER TABLE users ALTER COLUMN telegram_id DROP NOT NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS google_id            VARCHAR(255) UNIQUE,
    ADD COLUMN IF NOT EXISTS email                VARCHAR(255),
    ADD COLUMN IF NOT EXISTS telegram_link_code   VARCHAR(10),
    ADD COLUMN IF NOT EXISTS telegram_link_expires TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_google_id ON users (google_id) WHERE google_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_link_code ON users (telegram_link_code) WHERE telegram_link_code IS NOT NULL;

-- +goose Down
ALTER TABLE users ALTER COLUMN telegram_id SET NOT NULL;
ALTER TABLE users
    DROP COLUMN IF EXISTS google_id,
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS telegram_link_code,
    DROP COLUMN IF EXISTS telegram_link_expires;
