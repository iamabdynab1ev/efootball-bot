-- +goose Up
-- «Удалить чат у меня» (как в мессенджерах): точка очистки истории личного
-- диалога для одного участника. Сообщения с id <= upto_id для него скрыты,
-- диалог пропадает из списка, пока не придёт новое сообщение. Для собеседника
-- ничего не меняется. «Удалить у обоих» удаляет комнату целиком (CASCADE).
CREATE TABLE chat_clears (
    room_id    BIGINT NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    upto_id    BIGINT NOT NULL DEFAULT 0,
    cleared_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, user_id)
);

-- +goose Down
DROP TABLE chat_clears;
