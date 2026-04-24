-- +goose Up
-- +goose StatementBegin

ALTER TABLE users ADD COLUMN IF NOT EXISTS language VARCHAR(4) NOT NULL DEFAULT 'uz';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users DROP COLUMN IF EXISTS language;

-- +goose StatementEnd