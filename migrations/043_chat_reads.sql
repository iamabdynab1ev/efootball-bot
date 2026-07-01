-- +goose Up
-- +goose StatementBegin
-- Прогресс прочтения: до какого сообщения пользователь дочитал комнату.
-- Одна строка на (комната, пользователь). Отсюда считаем непрочитанные и
-- «галочки прочтения» (сравнивая last_read собеседника с id своих сообщений).
CREATE TABLE IF NOT EXISTS chat_reads (
    room_id      BIGINT NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_id BIGINT NOT NULL DEFAULT 0,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, user_id)
);

-- «Был(а) в сети»: обновляется при подключении/отключении SSE.
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN IF EXISTS last_seen_at;
DROP TABLE IF EXISTS chat_reads;
-- +goose StatementEnd
