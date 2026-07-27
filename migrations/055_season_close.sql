-- +goose Up
-- Закрытие сезона с церемонией: момент закрытия — для баннера «смотреть
-- церемонию» на главной и сортировки архива в Зале Славы.
ALTER TABLE seasons ADD COLUMN closed_at TIMESTAMPTZ;
UPDATE seasons SET closed_at = updated_at WHERE status = 'finished';

-- Сезонные номинации (league_id IS NULL) — по одной на сезон и тип:
-- повторное закрытие сезона идемпотентно обновляет, а не дублирует.
CREATE UNIQUE INDEX uq_season_awards_season_scope
  ON season_awards (season_id, award_type) WHERE league_id IS NULL;

-- +goose Down
DROP INDEX IF EXISTS uq_season_awards_season_scope;
ALTER TABLE seasons DROP COLUMN closed_at;
