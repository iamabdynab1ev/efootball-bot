-- +goose Up
-- +goose StatementBegin
-- Пометка «изменено» для отредактированных сообщений.
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS edited BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE chat_messages DROP COLUMN IF EXISTS edited;
-- +goose StatementEnd
