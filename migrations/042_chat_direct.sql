-- +goose Up
-- +goose StatementBegin
-- Личные сообщения (ЛС) переиспользуют chat_rooms/chat_messages: комната с
-- kind='direct' связывает ровно двух пользователей (dm_lo < dm_hi). Так весь
-- существующий конвейер сообщений (история/отправка/удаление/fan-out) работает
-- без изменений — меняется только контроль доступа и состав участников.
ALTER TABLE chat_rooms ADD COLUMN IF NOT EXISTS kind  TEXT   NOT NULL DEFAULT 'group';
ALTER TABLE chat_rooms ADD COLUMN IF NOT EXISTS dm_lo BIGINT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE chat_rooms ADD COLUMN IF NOT EXISTS dm_hi BIGINT REFERENCES users(id) ON DELETE CASCADE;

-- Групповые комнаты держат league_id; у ЛС его нет.
ALTER TABLE chat_rooms ALTER COLUMN league_id DROP NOT NULL;

-- Одна ЛС-комната на пару пользователей (нормализованная пара lo/hi).
CREATE UNIQUE INDEX IF NOT EXISTS uq_chat_dm ON chat_rooms (dm_lo, dm_hi) WHERE kind = 'direct';
-- Быстрый список диалогов пользователя (он в любой из двух позиций пары).
CREATE INDEX IF NOT EXISTS idx_chat_dm_lo ON chat_rooms (dm_lo) WHERE kind = 'direct';
CREATE INDEX IF NOT EXISTS idx_chat_dm_hi ON chat_rooms (dm_hi) WHERE kind = 'direct';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_chat_dm_hi;
DROP INDEX IF EXISTS idx_chat_dm_lo;
DROP INDEX IF EXISTS uq_chat_dm;
ALTER TABLE chat_rooms ALTER COLUMN league_id SET NOT NULL;
ALTER TABLE chat_rooms DROP COLUMN IF EXISTS dm_hi;
ALTER TABLE chat_rooms DROP COLUMN IF EXISTS dm_lo;
ALTER TABLE chat_rooms DROP COLUMN IF EXISTS kind;
-- +goose StatementEnd
