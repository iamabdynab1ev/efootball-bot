-- +goose Up
-- Настройки звука уведомлений (глобальный тумблер + по типам событий),
-- хранятся в профиле, чтобы переживать смену устройства/очистку кеша.
ALTER TABLE users ADD COLUMN sound_prefs JSONB;

-- +goose Down
ALTER TABLE users DROP COLUMN sound_prefs;
