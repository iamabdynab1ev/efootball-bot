-- +goose Up
-- «Получение» трофея: колонка claimed_at (NULL = ещё не забран игроком в
-- Трофейной комнате). Церемония получения теперь запускается вручную по кнопке
-- «Забрать», а не автоматически при выдаче — как распечатка награды.
ALTER TABLE user_achievements ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ;
ALTER TABLE season_awards     ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ;

-- Всё, что выдано ДО этой миграции, считаем уже полученным — иначе игрокам
-- прилетит лавина «заберите» за старые награды. Новые строки получают
-- claimed_at = NULL по умолчанию и становятся «неполученными».
UPDATE user_achievements SET claimed_at = earned_at  WHERE claimed_at IS NULL;
UPDATE season_awards     SET claimed_at = created_at WHERE claimed_at IS NULL;

-- +goose Down
ALTER TABLE user_achievements DROP COLUMN IF EXISTS claimed_at;
ALTER TABLE season_awards     DROP COLUMN IF EXISTS claimed_at;
