-- +goose Up
-- Делаем финализацию лиги идемпотентной: повторный вызов (авто после финала +
-- ручной из админки) обновляет награду вместо дублирования строки.
CREATE UNIQUE INDEX IF NOT EXISTS idx_season_awards_league_type
    ON season_awards(league_id, award_type)
    WHERE league_id IS NOT NULL;

-- Награды по лиге запрашиваются в hall-of-fame и admin — нужен индекс.
CREATE INDEX IF NOT EXISTS idx_season_awards_league ON season_awards(league_id);

-- bracket_slots.match_id ссылался на matches без ON DELETE — при удалении лиги
-- с уже сыгранным плей-офф каскад мог упереться в FK-ограничение.
ALTER TABLE bracket_slots DROP CONSTRAINT IF EXISTS bracket_slots_match_id_fkey;
ALTER TABLE bracket_slots
    ADD CONSTRAINT bracket_slots_match_id_fkey
    FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE bracket_slots DROP CONSTRAINT IF EXISTS bracket_slots_match_id_fkey;
ALTER TABLE bracket_slots
    ADD CONSTRAINT bracket_slots_match_id_fkey
    FOREIGN KEY (match_id) REFERENCES matches(id);
DROP INDEX IF EXISTS idx_season_awards_league;
DROP INDEX IF EXISTS idx_season_awards_league_type;
