-- +goose Up
-- +goose StatementBegin

ALTER TABLE users ADD COLUMN IF NOT EXISTS rank VARCHAR(32) NOT NULL DEFAULT '🥉 Новичок';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users DROP COLUMN IF EXISTS rank;

-- +goose StatementEnd