-- +goose Up
-- +goose StatementBegin

ALTER TABLE users ADD COLUMN IF NOT EXISTS rating INTEGER NOT NULL DEFAULT 1000;
ALTER TABLE users ADD COLUMN IF NOT EXISTS team_power INTEGER NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users DROP COLUMN IF EXISTS rating;
ALTER TABLE users DROP COLUMN IF EXISTS team_power;

-- +goose StatementEnd