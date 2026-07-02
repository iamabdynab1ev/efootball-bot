-- +goose Up
-- +goose StatementBegin
-- Медиа-вложение сообщения (голосовое/фото): {url, type, dur}. NULL — обычный текст.
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS media JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE chat_messages DROP COLUMN IF EXISTS media;
-- +goose StatementEnd
