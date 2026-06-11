-- +goose Up
-- Паритет живой БД со свежим деплоем (остатки потерянных миграций 18–25):
-- leagues.config нигде не используется кодом (единственное значение — '{}'),
-- matches.away_user_id в коде всегда обязателен (модель — int64, NULL-строк нет).
ALTER TABLE leagues DROP COLUMN IF EXISTS config;
ALTER TABLE matches ALTER COLUMN away_user_id SET NOT NULL;

-- +goose Down
ALTER TABLE matches ALTER COLUMN away_user_id DROP NOT NULL;
