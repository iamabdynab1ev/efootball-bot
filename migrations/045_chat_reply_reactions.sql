-- +goose Up
-- +goose StatementBegin
-- Ответ на сообщение: ссылка на исходное (в той же комнате). Превью ответа фронт
-- берёт из уже загруженных сообщений — доп. джоинов в выборке не требуется.
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS reply_to_id BIGINT REFERENCES chat_messages(id) ON DELETE SET NULL;

-- Реакции на сообщения (эмодзи). Одна реакция на (сообщение, пользователь, эмодзи).
CREATE TABLE IF NOT EXISTS chat_reactions (
    message_id BIGINT NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji      TEXT   NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji)
);
CREATE INDEX IF NOT EXISTS idx_chat_reactions_msg ON chat_reactions (message_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS chat_reactions;
ALTER TABLE chat_messages DROP COLUMN IF EXISTS reply_to_id;
-- +goose StatementEnd
