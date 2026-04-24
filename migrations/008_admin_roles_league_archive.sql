-- +goose Up
-- +goose StatementBegin

-- Admins жадвалига role устуни қўшамиз
ALTER TABLE admins ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'admin';
-- super_admin → ҳамма нарса (лига ўчириш, admin қўшиш/олиш)
-- admin       → аризалар, низолар, жеребьёвка

-- Leagues жадвалига archived статус қўшамиз
ALTER TYPE league_status ADD VALUE IF NOT EXISTS 'archived';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE admins DROP COLUMN IF EXISTS role;
-- +goose StatementEnd