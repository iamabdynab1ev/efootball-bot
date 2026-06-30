-- +goose Up
-- +goose StatementBegin
-- Чат турнира: по одной комнате на группу + общая комната лиги (group_name='').
-- Членство НЕ дублируется — выводится из league_members (approved + совпадение
-- группы), поэтому отдельной таблицы участников не нужно.
CREATE TABLE IF NOT EXISTS chat_rooms (
    id         BIGSERIAL PRIMARY KEY,
    league_id  BIGINT NOT NULL REFERENCES leagues(id) ON DELETE CASCADE,
    group_name TEXT   NOT NULL DEFAULT '', -- '' = общий чат лиги; 'A'/'B'/… = чат группы
    title      TEXT   NOT NULL,
    archived   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (league_id, group_name)
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id         BIGSERIAL PRIMARY KEY,
    room_id    BIGINT NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id    BIGINT REFERENCES users(id) ON DELETE SET NULL,
    body       TEXT   NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Лента комнаты и догрузка по id (история before / catch-up since) без seq-scan.
CREATE INDEX IF NOT EXISTS idx_chat_msg_room ON chat_messages (room_id, id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_rooms;
-- +goose StatementEnd
