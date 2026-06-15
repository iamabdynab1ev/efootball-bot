-- +goose Up

-- Best-of-X серии для матчей на выбывание. best_of=1 (по умолчанию) — обычный
-- одиночный матч, поведение не меняется. best_of>1 — серия: победитель тот, кто
-- первым выигрывает ceil(best_of/2) игр; home_wins/away_wins — текущий счёт серии.
ALTER TABLE leagues ADD COLUMN IF NOT EXISTS best_of   SMALLINT NOT NULL DEFAULT 1;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS best_of   SMALLINT NOT NULL DEFAULT 1;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS home_wins SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS away_wins SMALLINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE matches DROP COLUMN IF EXISTS away_wins;
ALTER TABLE matches DROP COLUMN IF EXISTS home_wins;
ALTER TABLE matches DROP COLUMN IF EXISTS best_of;
ALTER TABLE leagues DROP COLUMN IF EXISTS best_of;
